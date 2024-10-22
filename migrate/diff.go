package migrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

// Diff calculates the diff between the current database schema and the target state.
// The result changeset is not sorted, i.e. the caller should resolve dependencies
// before applying the changes.
func (d *detector) Diff() *changeset {
	targetTables := newTableSet(d.target.Tables...)
	currentTables := newTableSet(d.current.Tables...) // keeps state (which models still need to be checked)

	// These table-sets record changes to the targetTables set.
	created := newTableSet()
	renamed := newTableSet()

	// Discover CREATE/RENAME/DROP TABLE
	addedTables := targetTables.Sub(currentTables)
AddedLoop:
	for _, added := range addedTables.Values() {
		removedTables := currentTables.Sub(targetTables)
		for _, removed := range removedTables.Values() {
			if d.canRename(removed, added) {
				d.changes.Add(&RenameTable{
					FQN:     schema.FQN{removed.Schema, removed.Name},
					NewName: added.Name,
				})

				// Here we do not check for created / dropped columns, as well as column type changes,
				// because it is only possible to detect a renamed table if its signature (see state.go) did not change.
				d.detectColumnChanges(removed, added, false)
				d.detectConstraintChanges(removed, added)

				// Update referenced table in all related FKs.
				if d.detectRenamedFKs {
					d.refMap.UpdateT(removed.T(), added.T())
				}

				renamed.Add(added)

				// Do not check this model further, we know it was renamed.
				currentTables.Remove(removed.Name)
				continue AddedLoop
			}
		}
		// If a new table did not appear because of the rename operation, then it must've been created.
		d.changes.Add(&CreateTable{
			FQN:   schema.FQN{added.Schema, added.Name},
			Model: added.Model,
		})
		created.Add(added)
	}

	// Tables that aren't present anymore and weren't renamed or left untouched were deleted.
	dropped := currentTables.Sub(targetTables)
	for _, t := range dropped.Values() {
		d.changes.Add(&DropTable{
			FQN: schema.FQN{t.Schema, t.Name},
		})
	}

	// Detect changes in existing tables that weren't renamed.
	//
	// TODO: here having State.Tables be a map[string]Table would be much more convenient.
	// Then we can alse retire tableSet, or at least simplify it to a certain extent.
	curEx := currentTables.Sub(dropped)
	tarEx := targetTables.Sub(created).Sub(renamed)
	for _, target := range tarEx.Values() {
		// TODO(dyma): step is redundant if we have map[string]Table
		var current sqlschema.Table
		for _, cur := range curEx.Values() {
			if cur.Name == target.Name {
				current = cur
				break
			}
		}
		d.detectColumnChanges(current, target, true)
		d.detectConstraintChanges(current, target)
	}

	// Compare and update FKs ----------------
	currentFKs := make(map[sqlschema.FK]string)
	for k, v := range d.current.FKs {
		currentFKs[k] = v
	}

	if d.detectRenamedFKs {
		// Add RenameFK migrations for updated FKs.
		for old, renamed := range d.refMap.Updated() {
			newName := d.fkNameFunc(renamed)
			d.changes.Add(&RenameConstraint{
				FK:      renamed, // TODO: make sure this is applied after the table/columns are renamed
				OldName: d.current.FKs[old],
				NewName: newName,
			})

			// Add this FK to currentFKs to prevent it from firing in the two loops below.
			currentFKs[renamed] = newName
			delete(currentFKs, old)
		}
	}

	// Add AddFK migrations for newly added FKs.
	for fk := range d.target.FKs {
		if _, ok := currentFKs[fk]; !ok {
			d.changes.Add(&AddForeignKey{
				FK:             fk,
				ConstraintName: d.fkNameFunc(fk),
			})
		}
	}

	// Add DropFK migrations for removed FKs.
	for fk, fkName := range currentFKs {
		if _, ok := d.target.FKs[fk]; !ok {
			d.changes.Add(&DropConstraint{
				FK:             fk,
				ConstraintName: fkName,
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
		var operations []interface{}
		for _, op := range c.operations {
			operations = append(operations, op.(interface{}))
		}
		return m.Apply(ctx, operations...)
	}
}

// Up is syntactic sugar.
func (c *changeset) Up(m sqlschema.Migrator) MigrationFunc {
	return c.Func(m)
}

// Down is syntactic sugar.
func (c *changeset) Down(m sqlschema.Migrator) MigrationFunc {
	var reverse changeset
	for i := len(c.operations) - 1; i >= 0; i-- {
		reverse.Add(c.operations[i].GetReverse())
	}
	return reverse.Func(m)
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

func withFKNameFunc(f func(sqlschema.FK) string) diffOption {
	return func(cfg *detectorConfig) {
		cfg.FKNameFunc = f
	}
}

func withDetectRenamedFKs(enabled bool) diffOption {
	return func(cfg *detectorConfig) {
		cfg.DetectRenamedFKs = enabled
	}
}

func withTypeEquivalenceFunc(f sqlschema.TypeEquivalenceFunc) diffOption {
	return func(cfg *detectorConfig) {
		cfg.EqType = f
	}
}

// detectorConfig controls how differences in the model states are resolved.
type detectorConfig struct {
	FKNameFunc       func(sqlschema.FK) string
	DetectRenamedFKs bool
	EqType           sqlschema.TypeEquivalenceFunc
}

type detector struct {
	// current state represents the existing database schema.
	current sqlschema.State

	// target state represents the database schema defined in bun models.
	target sqlschema.State

	changes changeset
	refMap  sqlschema.RefMap

	// fkNameFunc builds the name for created/renamed FK contraints.
	fkNameFunc func(sqlschema.FK) string

	// eqType determines column type equivalence.
	// Default is direct comparison with '==' operator, which is inaccurate
	// due to the existence of dialect-specific type aliases. The caller
	// should pass a concrete InspectorDialect.EquuivalentType for robust comparison.
	eqType sqlschema.TypeEquivalenceFunc

	// detectRenemedFKs controls how FKs are treated when their references (table/column) are renamed.
	detectRenamedFKs bool
}

func newDetector(got, want sqlschema.State, opts ...diffOption) *detector {
	cfg := &detectorConfig{
		FKNameFunc:       defaultFKName,
		DetectRenamedFKs: false,
		EqType: func(c1, c2 sqlschema.Column) bool {
			return c1.SQLType == c2.SQLType && c1.VarcharLen == c2.VarcharLen
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var existingFKs []sqlschema.FK
	for fk := range got.FKs {
		existingFKs = append(existingFKs, fk)
	}

	return &detector{
		current:          got,
		target:           want,
		refMap:           sqlschema.NewRefMap(existingFKs...),
		fkNameFunc:       cfg.FKNameFunc,
		detectRenamedFKs: cfg.DetectRenamedFKs,
		eqType:           cfg.EqType,
	}
}

// canRename checks if t1 can be renamed to t2.
func (d *detector) canRename(t1, t2 sqlschema.Table) bool {
	return t1.Schema == t2.Schema && sqlschema.EqualSignatures(t1, t2, d.equalColumns)
}

func (d *detector) equalColumns(col1, col2 sqlschema.Column) bool {
	return d.eqType(col1, col2) &&
		col1.DefaultValue == col2.DefaultValue &&
		col1.IsNullable == col2.IsNullable &&
		col1.IsAutoIncrement == col2.IsAutoIncrement &&
		col1.IsIdentity == col2.IsIdentity
}

func (d *detector) makeTargetColDef(current, target sqlschema.Column) sqlschema.Column {
	// Avoid unneccessary type-change migrations if the types are equivalent.
	if d.eqType(current, target) {
		target.SQLType = current.SQLType
		target.VarcharLen = current.VarcharLen
	}
	return target
}

// detechColumnChanges finds renamed columns and, if checkType == true, columns with changed type.
func (d *detector) detectColumnChanges(current, target sqlschema.Table, checkType bool) {
	fqn := schema.FQN{target.Schema, target.Name}

ChangedRenamed:
	for tName, tCol := range target.Columns {

		// This column exists in the database, so it hasn't been renamed, dropped, or added.
		// Still, we should not delete(columns, thisColumn), because later we will need to
		// check that we do not try to rename a column to an already a name that already exists.
		if cCol, ok := current.Columns[tName]; ok {
			if checkType && !d.equalColumns(cCol, tCol) {
				d.changes.Add(&ChangeColumnType{
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
		for cName, cCol := range current.Columns {
			// Cannot rename if a column with this name already exists or the types differ.
			if _, exists := target.Columns[cName]; exists || !d.equalColumns(tCol, cCol) {
				continue
			}
			d.changes.Add(&RenameColumn{
				FQN:     fqn,
				OldName: cName,
				NewName: tName,
			})
			delete(current.Columns, cName) // no need to check this column again
			d.refMap.UpdateC(sqlschema.C(target.Schema, target.Name, cName), tName)

			continue ChangedRenamed
		}

		d.changes.Add(&AddColumn{
			FQN:    fqn,
			Column: tName,
			ColDef: tCol,
		})
	}

	// Drop columns which do not exist in the target schema and were not renamed.
	for cName, cCol := range current.Columns {
		if _, keep := target.Columns[cName]; !keep {
			d.changes.Add(&DropColumn{
				FQN:    fqn,
				Column: cName,
				ColDef: cCol,
			})
		}
	}
}

func (d *detector) detectConstraintChanges(current, target sqlschema.Table) {
	fqn := schema.FQN{target.Schema, target.Name}

Add:
	for _, want := range target.UniqueContraints {
		for _, got := range current.UniqueContraints {
			if got.Equals(want) {
				continue Add
			}
		}
		d.changes.Add(&AddUniqueConstraint{
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

		d.changes.Add(&DropUniqueConstraint{
			FQN:    fqn,
			Unique: got,
		})
	}
}

// sqlschema utils ------------------------------------------------------------

// tableSet stores unique table definitions.
type tableSet struct {
	underlying map[string]sqlschema.Table
}

func newTableSet(initial ...sqlschema.Table) *tableSet {
	set := &tableSet{
		underlying: make(map[string]sqlschema.Table),
	}
	for _, t := range initial {
		set.Add(t)
	}
	return set
}

func (set *tableSet) Add(t sqlschema.Table) {
	set.underlying[t.Name] = t
}

func (set *tableSet) Remove(s string) {
	delete(set.underlying, s)
}

func (set *tableSet) Values() (tables []sqlschema.Table) {
	for _, t := range set.underlying {
		tables = append(tables, t)
	}
	return
}

func (set *tableSet) Sub(other *tableSet) *tableSet {
	res := set.clone()
	for v := range other.underlying {
		if _, ok := set.underlying[v]; ok {
			res.Remove(v)
		}
	}
	return res
}

func (set *tableSet) clone() *tableSet {
	res := newTableSet()
	for _, t := range set.underlying {
		res.Add(t)
	}
	return res
}

// String is a debug helper to get a list of table names in the set.
func (set *tableSet) String() string {
	var s strings.Builder
	for k := range set.underlying {
		if s.Len() > 0 {
			s.WriteString(", ")
		}
		s.WriteString(k)
	}
	return s.String()
}

// defaultFKName returns a name for the FK constraint in the format {tablename}_{columnname(s)}_fkey, following the Postgres convention.
func defaultFKName(fk sqlschema.FK) string {
	columnnames := strings.Join(fk.From.Column.Split(), "_")
	return fmt.Sprintf("%s_%s_fkey", fk.From.Table, columnnames)
}
