package migrate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

// Diff calculates the diff between the current database schema and the target state.
// The result changeset is not sorted, i.e. the caller should resolve dependencies
// before applying the changes.
func (d *detector) Diff() *changeset {
RenameCreate:
	for wantName, wantTable := range d.target.TableDefinitions {

		// A table with this name exists in the database. We assume that schema objects won't
		// be renamed to an already existing name, nor do we support such cases.
		// Simply check if the table definition has changed.
		if haveTable, ok := d.current.TableDefinitions[wantName]; ok {
			d.detectColumnChanges(haveTable, wantTable, true)
			d.detectConstraintChanges(haveTable, wantTable)
			continue
		}

		// Find all renamed tables. We assume that renamed tables have the same signature.
		for haveName, haveTable := range d.current.TableDefinitions {
			if _, exists := d.target.TableDefinitions[haveName]; !exists && d.canRename(haveTable, wantTable) {
				d.changes.Add(&RenameTableOp{
					FQN:     haveTable.FQN(),
					NewName: wantName,
				})
				d.refMap.RenameTable(haveTable.FQN(), wantName)

				// Find renamed columns, if any, and check if constraints (PK, UNIQUE) have been updated.
				// We need not check wantTable any further.
				d.detectColumnChanges(haveTable, wantTable, false)
				d.detectConstraintChanges(haveTable, wantTable)
				delete(d.current.TableDefinitions, haveName)
				continue RenameCreate
			}
		}

		// If wantTable does not exist in the database and was not renamed
		// then we need to create this table in the database.
		additional := wantTable.Additional.(sqlschema.SchemaTable)
		d.changes.Add(&CreateTableOp{
			FQN:   wantTable.FQN(),
			Model: additional.Model,
		})
	}

	// Drop any remaining "current" tables which do not have a model.
	for name, table := range d.current.TableDefinitions {
		if _, keep := d.target.TableDefinitions[name]; !keep {
			d.changes.Add(&DropTableOp{
				FQN: table.FQN(),
			})
		}
	}

	currentFKs := d.refMap.Deref()

	for fk := range d.target.ForeignKeys {
		if _, ok := currentFKs[fk]; !ok {
			d.changes.Add(&AddForeignKeyOp{
				ForeignKey:     fk,
				ConstraintName: d.fkNameFunc(fk),
			})
		}
	}

	for fk, name := range currentFKs {
		if _, ok := d.target.ForeignKeys[fk]; !ok {
			d.changes.Add(&DropForeignKeyOp{
				ConstraintName: name,
				ForeignKey:     fk,
			})
		}
	}

	return &d.changes
}

// changeset is a set of changes to the database schema definition.
type changeset struct {
	operations []Operation
}

// Add new operations to the changeset.
func (c *changeset) Add(op ...Operation) {
	c.operations = append(c.operations, op...)
}

// Func creates a MigrationFunc that applies all operations all the changeset.
func (c *changeset) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return c.apply(ctx, db, m)
	}
}

// GetReverse returns a new changeset with each operation in it "reversed" and in reverse order.
func (c *changeset) GetReverse() *changeset {
	var reverse changeset
	for i := len(c.operations) - 1; i >= 0; i-- {
		reverse.Add(c.operations[i].GetReverse())
	}
	return &reverse
}

// Up is syntactic sugar.
func (c *changeset) Up(m sqlschema.Migrator) MigrationFunc {
	return c.Func(m)
}

// Down is syntactic sugar.
func (c *changeset) Down(m sqlschema.Migrator) MigrationFunc {
	return c.GetReverse().Func(m)
}

// apply generates SQL for each operation and executes it.
func (c *changeset) apply(ctx context.Context, db *bun.DB, m sqlschema.Migrator) error {
	if len(c.operations) == 0 {
		return nil
	}

	for _, op := range c.operations {
		if _, isComment := op.(*comment); isComment {
			continue
		}

		b := internal.MakeQueryBytes()
		b, err := m.AppendSQL(b, op)
		if err != nil {
			return fmt.Errorf("apply changes: %w", err)
		}

		query := internal.String(b)
		if _, err = db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("apply changes: %w", err)
		}
	}
	return nil
}

func (c *changeset) WriteTo(w io.Writer, m sqlschema.Migrator) error {
	var err error

	b := internal.MakeQueryBytes()
	for _, op := range c.operations {
		if c, isComment := op.(*comment); isComment {
			b = append(b, "/*\n"...)
			b = append(b, *c...)
			b = append(b, "\n*/"...)
			continue
		}

		b, err = m.AppendSQL(b, op)
		if err != nil {
			return fmt.Errorf("write changeset: %w", err)
		}
		b = append(b, ";\n"...)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("write changeset: %w", err)
	}
	return nil
}

func (c *changeset) ResolveDependencies() error {
	if len(c.operations) <= 1 {
		return nil
	}

	const (
		unvisited = iota
		current
		visited
	)

	var resolved []Operation
	var visit func(op Operation) error

	var nextOp Operation
	var next func() bool

	status := make(map[Operation]int, len(c.operations))
	for _, op := range c.operations {
		status[op] = unvisited
	}

	next = func() bool {
		for op, s := range status {
			if s == unvisited {
				nextOp = op
				return true
			}
		}
		return false
	}

	// visit iterates over c.operations until it finds all operations that depend on the current one
	// or runs into cirtular dependency, in which case it will return an error.
	visit = func(op Operation) error {
		switch status[op] {
		case visited:
			return nil
		case current:
			// TODO: add details (circle) to the error message
			return errors.New("detected circular dependency")
		}

		status[op] = current

		for _, another := range c.operations {
			if dop, hasDeps := another.(interface {
				DependsOn(Operation) bool
			}); another == op || !hasDeps || !dop.DependsOn(op) {
				continue
			}
			if err := visit(another); err != nil {
				return err
			}
		}

		status[op] = visited

		// Any dependent nodes would've already been added to the list by now, so we prepend.
		resolved = append([]Operation{op}, resolved...)
		return nil
	}

	for next() {
		if err := visit(nextOp); err != nil {
			return err
		}
	}

	c.operations = resolved
	return nil
}

type diffOption func(*detectorConfig)

func withFKNameFunc(f func(sqlschema.ForeignKey) string) diffOption {
	return func(cfg *detectorConfig) {
		// cfg.FKNameFunc = f
	}
}

func withDetectRenamedFKs(enabled bool) diffOption {
	return func(cfg *detectorConfig) {
		cfg.DetectRenamedFKs = enabled
	}
}

func withTypeEquivalenceFunc(f TypeEquivalenceFunc) diffOption {
	return func(cfg *detectorConfig) {
		cfg.EqType = f
	}
}

// detectorConfig controls how differences in the model states are resolved.
type detectorConfig struct {
	FKNameFunc       func(sqlschema.ForeignKey) string
	DetectRenamedFKs bool
	EqType           TypeEquivalenceFunc
}

// detector may modify the passed database schemas, so it isn't safe to re-use them.
type detector struct {
	// current state represents the existing database schema.
	current sqlschema.DatabaseSchema

	// target state represents the database schema defined in bun models.
	target sqlschema.DatabaseSchema

	changes changeset
	refMap  refMap

	// fkNameFunc builds the name for created/renamed FK contraints.
	fkNameFunc func(sqlschema.ForeignKey) string

	// eqType determines column type equivalence.
	// Default is direct comparison with '==' operator, which is inaccurate
	// due to the existence of dialect-specific type aliases. The caller
	// should pass a concrete InspectorDialect.EquuivalentType for robust comparison.
	eqType TypeEquivalenceFunc

	// detectRenemedFKs controls how FKs are treated when their references (table/column) are renamed.
	detectRenamedFKs bool
}

func newDetector(got, want sqlschema.DatabaseSchema, opts ...diffOption) *detector {
	cfg := &detectorConfig{
		FKNameFunc:       defaultFKName,
		DetectRenamedFKs: false,
		EqType: func(c1, c2 sqlschema.ColumnDefinition) bool {
			return c1.SQLType == c2.SQLType && c1.VarcharLen == c2.VarcharLen
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &detector{
		current:          got,
		target:           want,
		refMap:           newRefMap(got.ForeignKeys),
		fkNameFunc:       cfg.FKNameFunc,
		detectRenamedFKs: cfg.DetectRenamedFKs,
		eqType:           cfg.EqType,
	}
}

// canRename checks if t1 can be renamed to t2.
func (d *detector) canRename(t1, t2 sqlschema.TableDefinition) bool {
	return t1.Schema == t2.Schema && equalSignatures(t1, t2, d.equalColumns)
}

func (d *detector) equalColumns(col1, col2 sqlschema.ColumnDefinition) bool {
	return d.eqType(col1, col2) &&
		col1.DefaultValue == col2.DefaultValue &&
		col1.IsNullable == col2.IsNullable &&
		col1.IsAutoIncrement == col2.IsAutoIncrement &&
		col1.IsIdentity == col2.IsIdentity
}

func (d *detector) makeTargetColDef(current, target sqlschema.ColumnDefinition) sqlschema.ColumnDefinition {
	// Avoid unneccessary type-change migrations if the types are equivalent.
	if d.eqType(current, target) {
		target.SQLType = current.SQLType
		target.VarcharLen = current.VarcharLen
	}
	return target
}

// detechColumnChanges finds renamed columns and, if checkType == true, columns with changed type.
func (d *detector) detectColumnChanges(current, target sqlschema.TableDefinition, checkType bool) {
	fqn := schema.FQN{Schema: target.Schema, Table: target.Name}

ChangedRenamed:
	for tName, tCol := range target.ColumnDefimitions {

		// This column exists in the database, so it hasn't been renamed, dropped, or added.
		// Still, we should not delete(columns, thisColumn), because later we will need to
		// check that we do not try to rename a column to an already a name that already exists.
		if cCol, ok := current.ColumnDefimitions[tName]; ok {
			if checkType && !d.equalColumns(cCol, tCol) {
				d.changes.Add(&ChangeColumnTypeOp{
					FQN:    fqn,
					Column: tName,
					From:   cCol,
					To:     d.makeTargetColDef(cCol, tCol),
				})
			}
			continue
		}

		// Column tName does not exist in the database -- it's been either renamed or added.
		// Find renamed columns first.
		for cName, cCol := range current.ColumnDefimitions {
			// Cannot rename if a column with this name already exists or the types differ.
			if _, exists := target.ColumnDefimitions[cName]; exists || !d.equalColumns(tCol, cCol) {
				continue
			}
			d.changes.Add(&RenameColumnOp{
				FQN:     fqn,
				OldName: cName,
				NewName: tName,
			})
			d.refMap.RenameColumn(fqn, cName, tName)
			delete(current.ColumnDefimitions, cName) // no need to check this column again

			// Update primary key definition to avoid superficially recreating the constraint.
			current.PrimaryKey.Columns.Replace(cName, tName)

			continue ChangedRenamed
		}

		d.changes.Add(&AddColumnOp{
			FQN:    fqn,
			Column: tName,
			ColDef: tCol,
		})
	}

	// Drop columns which do not exist in the target schema and were not renamed.
	for cName, cCol := range current.ColumnDefimitions {
		if _, keep := target.ColumnDefimitions[cName]; !keep {
			d.changes.Add(&DropColumnOp{
				FQN:    fqn,
				Column: cName,
				ColDef: cCol,
			})
		}
	}
}

func (d *detector) detectConstraintChanges(current, target sqlschema.TableDefinition) {
	fqn := schema.FQN{Schema: target.Schema, Table: target.Name}

Add:
	for _, want := range target.UniqueContraints {
		for _, got := range current.UniqueContraints {
			if got.Equals(want) {
				continue Add
			}
		}
		d.changes.Add(&AddUniqueConstraintOp{
			FQN:    fqn,
			Unique: want,
		})
	}

Drop:
	for _, got := range current.UniqueContraints {
		for _, want := range target.UniqueContraints {
			if got.Equals(want) {
				continue Drop
			}
		}

		d.changes.Add(&DropUniqueConstraintOp{
			FQN:    fqn,
			Unique: got,
		})
	}

	// Detect primary key changes
	if target.PrimaryKey == nil && current.PrimaryKey == nil {
		return
	}
	switch {
	case target.PrimaryKey == nil && current.PrimaryKey != nil:
		d.changes.Add(&DropPrimaryKeyOp{
			FQN:        fqn,
			PrimaryKey: *current.PrimaryKey,
		})
	case current.PrimaryKey == nil && target.PrimaryKey != nil:
		d.changes.Add(&AddPrimaryKeyOp{
			FQN:        fqn,
			PrimaryKey: *target.PrimaryKey,
		})
	case target.PrimaryKey.Columns != current.PrimaryKey.Columns:
		d.changes.Add(&ChangePrimaryKeyOp{
			FQN: fqn,
			Old: *current.PrimaryKey,
			New: *target.PrimaryKey,
		})
	}
}

// defaultFKName returns a name for the FK constraint in the format {tablename}_{columnname(s)}_fkey, following the Postgres convention.
func defaultFKName(fk sqlschema.ForeignKey) string {
	columnnames := strings.Join(fk.From.Column.Split(), "_")
	return fmt.Sprintf("%s_%s_fkey", fk.From.FQN.Table, columnnames)
}

type TypeEquivalenceFunc func(sqlschema.ColumnDefinition, sqlschema.ColumnDefinition) bool

// equalSignatures determines if two tables have the same "signature".
func equalSignatures(t1, t2 sqlschema.TableDefinition, eq TypeEquivalenceFunc) bool {
	sig1 := newSignature(t1, eq)
	sig2 := newSignature(t2, eq)
	return sig1.Equals(sig2)
}

// signature is a set of column definitions, which allows "relation/name-agnostic" comparison between them;
// meaning that two columns are considered equal if their types are the same.
type signature struct {

	// underlying stores the number of occurences for each unique column type.
	// It helps to account for the fact that a table might have multiple columns that have the same type.
	underlying map[sqlschema.ColumnDefinition]int

	eq TypeEquivalenceFunc
}

func newSignature(t sqlschema.TableDefinition, eq TypeEquivalenceFunc) signature {
	s := signature{
		underlying: make(map[sqlschema.ColumnDefinition]int),
		eq:         eq,
	}
	s.scan(t)
	return s
}

// scan iterates over table's field and counts occurrences of each unique column definition.
func (s *signature) scan(t sqlschema.TableDefinition) {
	for _, scanCol := range t.ColumnDefimitions {
		// This is slightly more expensive than if the columns could be compared directly
		// and we always did s.underlying[col]++, but we get type-equivalence in return.
		col, count := s.getCount(scanCol)
		if count == 0 {
			s.underlying[scanCol] = 1
		} else {
			s.underlying[col]++
		}
	}
}

// getCount uses TypeEquivalenceFunc to find a column with the same (equivalent) SQL type
// and returns its count. Count 0 means there are no columns with of this type.
func (s *signature) getCount(keyCol sqlschema.ColumnDefinition) (key sqlschema.ColumnDefinition, count int) {
	for col, cnt := range s.underlying {
		if s.eq(col, keyCol) {
			return col, cnt
		}
	}
	return keyCol, 0
}

// Equals returns true if 2 signatures share an identical set of columns.
func (s *signature) Equals(other signature) bool {
	if len(s.underlying) != len(other.underlying) {
		return false
	}
	for col, count := range s.underlying {
		if _, countOther := other.getCount(col); countOther != count {
			return false
		}
	}
	return true
}

type refMap map[*sqlschema.ForeignKey]string

func newRefMap(fks map[sqlschema.ForeignKey]string) refMap {
	rm := make(map[*sqlschema.ForeignKey]string)
	for fk, name := range fks {
		rm[&fk] = name
	}
	return rm
}

func (rm refMap) RenameTable(table schema.FQN, newName string) {
	for fk := range rm {
		switch table {
		case fk.From.FQN:
			fk.From.FQN.Table = newName
		case fk.To.FQN:
			fk.To.FQN.Table = newName
		}
	}
}

func (rm refMap) RenameColumn(table schema.FQN, column, newName string) {
	for fk := range rm {
		if table == fk.From.FQN {
			fk.From.Column.Replace(column, newName)
		}
		if table == fk.To.FQN {
			fk.To.Column.Replace(column, newName)
		}
	}
}

func (rm refMap) Deref() map[sqlschema.ForeignKey]string {
	out := make(map[sqlschema.ForeignKey]string)
	for fk, name := range rm {
		out[*fk] = name
	}
	return out
}
