package migrate

import (
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

// changeset is a set of changes to the database schema definition.
type changeset struct {
	operations []Operation
}

// Add new operations to the changeset.
func (c *changeset) Add(op ...Operation) {
	c.operations = append(c.operations, op...)
}

// diff calculates the diff between the current database schema and the target state.
// The changeset is not sorted -- the caller should resolve dependencies before applying the changes.
func diff(got, want sqlschema.Schema, opts ...diffOption) *changeset {
	d := newDetector(got, want, opts...)
	return d.detectChanges()
}

func (d *detector) detectChanges() *changeset {
	currentTables := d.mapNameToTable(d.current)
	targetTables := d.mapNameToTable(d.target)

RenameCreate:
	for wantName, wantTable := range targetTables {

		// A table with this name exists in the database. We assume that schema objects won't
		// be renamed to an already existing name, nor do we support such cases.
		// Simply check if the table definition has changed.
		if haveTable, ok := currentTables[wantName]; ok {
			d.detectColumnChanges(haveTable, wantTable, true)
			d.detectConstraintChanges(haveTable, wantTable)
			continue
		}

		// Find all renamed tables. We assume that renamed tables have the same signature.
		for haveName, haveTable := range currentTables {
			if _, exists := targetTables[haveName]; !exists && d.canRename(haveTable, wantTable) {
				d.changes.Add(&RenameTableOp{
					FQN:     haveTable.GetFQN(),
					NewName: wantName,
				})
				d.refMap.RenameTable(haveTable.GetFQN(), wantName)

				// Find renamed columns, if any, and check if constraints (PK, UNIQUE) have been updated.
				// We need not check wantTable any further.
				d.detectColumnChanges(haveTable, wantTable, false)
				d.detectConstraintChanges(haveTable, wantTable)
				delete(currentTables, haveName)
				continue RenameCreate
			}
		}

		// If wantTable does not exist in the database and was not renamed
		// then we need to create this table in the database.
		additional := wantTable.(sqlschema.BunTable)
		d.changes.Add(&CreateTableOp{
			FQN:   wantTable.GetFQN(),
			Model: additional.Model,
		})
	}

	// Drop any remaining "current" tables which do not have a model.
	for name, table := range currentTables {
		if _, keep := targetTables[name]; !keep {
			d.changes.Add(&DropTableOp{
				FQN: table.GetFQN(),
			})
		}
	}

	targetFKs := d.target.GetForeignKeys()
	currentFKs := d.refMap.Deref()

	for fk := range targetFKs {
		if _, ok := currentFKs[fk]; !ok {
			d.changes.Add(&AddForeignKeyOp{
				ForeignKey:     fk,
				ConstraintName: "", // leave empty to let each dialect apply their convention
			})
		}
	}

	for fk, name := range currentFKs {
		if _, ok := targetFKs[fk]; !ok {
			d.changes.Add(&DropForeignKeyOp{
				ConstraintName: name,
				ForeignKey:     fk,
			})
		}
	}

	return &d.changes
}

// detechColumnChanges finds renamed columns and, if checkType == true, columns with changed type.
func (d *detector) detectColumnChanges(current, target sqlschema.Table, checkType bool) {
	currentColumns := d.mapNameToColumn(current)
	targetColumns := d.mapNameToColumn(target)

ChangeRename:
	for tName, tCol := range targetColumns {

		// This column exists in the database, so it hasn't been renamed, dropped, or added.
		// Still, we should not delete(columns, thisColumn), because later we will need to
		// check that we do not try to rename a column to an already a name that already exists.
		if cCol, ok := currentColumns[tName]; ok {
			if checkType && !d.equalColumns(cCol, tCol) {
				d.changes.Add(&ChangeColumnTypeOp{
					FQN:    target.GetFQN(),
					Column: tName,
					From:   cCol,
					To:     d.makeTargetColDef(cCol, tCol),
				})
			}
			continue
		}

		// Column tName does not exist in the database -- it's been either renamed or added.
		// Find renamed columns first.
		for cName, cCol := range currentColumns {
			// Cannot rename if a column with this name already exists or the types differ.
			if _, exists := targetColumns[cName]; exists || !d.equalColumns(tCol, cCol) {
				continue
			}
			d.changes.Add(&RenameColumnOp{
				FQN:     target.GetFQN(),
				OldName: cName,
				NewName: tName,
			})
			d.refMap.RenameColumn(target.GetFQN(), cName, tName)
			delete(currentColumns, cName) // no need to check this column again

			// Update primary key definition to avoid superficially recreating the constraint.
			current.GetPrimaryKey().Columns.Replace(cName, tName)

			continue ChangeRename
		}

		d.changes.Add(&AddColumnOp{
			FQN:    target.GetFQN(),
			Column: tName,
			ColDef: tCol,
		})
	}

	// Drop columns which do not exist in the target schema and were not renamed.
	for cName, cCol := range currentColumns {
		if _, keep := targetColumns[cName]; !keep {
			d.changes.Add(&DropColumnOp{
				FQN:    target.GetFQN(),
				Column: cName,
				ColDef: cCol,
			})
		}
	}
}

func (d *detector) detectConstraintChanges(current, target sqlschema.Table) {
Add:
	for _, want := range target.GetUniqueConstraints() {
		for _, got := range current.GetUniqueConstraints() {
			if got.Equals(want) {
				continue Add
			}
		}
		d.changes.Add(&AddUniqueConstraintOp{
			FQN:    target.GetFQN(),
			Unique: want,
		})
	}

Drop:
	for _, got := range current.GetUniqueConstraints() {
		for _, want := range target.GetUniqueConstraints() {
			if got.Equals(want) {
				continue Drop
			}
		}

		d.changes.Add(&DropUniqueConstraintOp{
			FQN:    target.GetFQN(),
			Unique: got,
		})
	}

	targetPK := target.GetPrimaryKey()
	currentPK := current.GetPrimaryKey()

	// Detect primary key changes
	if targetPK == nil && currentPK == nil {
		return
	}
	switch {
	case targetPK == nil && currentPK != nil:
		d.changes.Add(&DropPrimaryKeyOp{
			FQN:        target.GetFQN(),
			PrimaryKey: *currentPK,
		})
	case currentPK == nil && targetPK != nil:
		d.changes.Add(&AddPrimaryKeyOp{
			FQN:        target.GetFQN(),
			PrimaryKey: *targetPK,
		})
	case targetPK.Columns != currentPK.Columns:
		d.changes.Add(&ChangePrimaryKeyOp{
			FQN: target.GetFQN(),
			Old: *currentPK,
			New: *targetPK,
		})
	}
}

func newDetector(got, want sqlschema.Schema, opts ...diffOption) *detector {
	cfg := &detectorConfig{
		EqType: func(c1, c2 sqlschema.Column) bool {
			return c1.GetSQLType() == c2.GetSQLType() && c1.GetVarcharLen() == c2.GetVarcharLen()
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &detector{
		current: got,
		target:  want,
		refMap:  newRefMap(got.GetForeignKeys()),
		eqType:  cfg.EqType,
	}
}

type diffOption func(*detectorConfig)

func withTypeEquivalenceFunc(f TypeEquivalenceFunc) diffOption {
	return func(cfg *detectorConfig) {
		cfg.EqType = f
	}
}

// detectorConfig controls how differences in the model states are resolved.
type detectorConfig struct {
	EqType TypeEquivalenceFunc
}

// detector may modify the passed database schemas, so it isn't safe to re-use them.
type detector struct {
	// current state represents the existing database schema.
	current sqlschema.Schema

	// target state represents the database schema defined in bun models.
	target sqlschema.Schema

	changes changeset
	refMap  refMap

	// eqType determines column type equivalence.
	// Default is direct comparison with '==' operator, which is inaccurate
	// due to the existence of dialect-specific type aliases. The caller
	// should pass a concrete InspectorDialect.EquuivalentType for robust comparison.
	eqType TypeEquivalenceFunc
}

// canRename checks if t1 can be renamed to t2.
func (d detector) canRename(t1, t2 sqlschema.Table) bool {
	return t1.GetSchema() == t2.GetSchema() && equalSignatures(t1, t2, d.equalColumns)
}

func (d detector) equalColumns(col1, col2 sqlschema.Column) bool {
	return d.eqType(col1, col2) &&
		col1.GetDefaultValue() == col2.GetDefaultValue() &&
		col1.GetIsNullable() == col2.GetIsNullable() &&
		col1.GetIsAutoIncrement() == col2.GetIsAutoIncrement() &&
		col1.GetIsIdentity() == col2.GetIsIdentity()
}

func (d detector) makeTargetColDef(current, target sqlschema.Column) sqlschema.Column {
	// Avoid unneccessary type-change migrations if the types are equivalent.
	if d.eqType(current, target) {
		target = sqlschema.ColumnDefinition{
			Name:            target.GetName(),
			DefaultValue:    target.GetDefaultValue(),
			IsNullable:      target.GetIsNullable(),
			IsAutoIncrement: target.GetIsAutoIncrement(),
			IsIdentity:      target.GetIsIdentity(),

			SQLType:    current.GetSQLType(),
			VarcharLen: current.GetVarcharLen(),
		}
	}
	return target
}

func (d *detector) mapNameToTable(s sqlschema.Schema) map[string]sqlschema.Table {
	m := make(map[string]sqlschema.Table)
	for _, t := range s.GetTables() {
		m[t.GetName()] = t
	}
	return m
}

func (d *detector) mapNameToColumn(t sqlschema.Table) map[string]sqlschema.Column {
	m := make(map[string]sqlschema.Column)
	for _, c := range t.GetColumns() {
		m[c.GetName()] = c
	}
	return m
}

type TypeEquivalenceFunc func(sqlschema.Column, sqlschema.Column) bool

// equalSignatures determines if two tables have the same "signature".
func equalSignatures(t1, t2 sqlschema.Table, eq TypeEquivalenceFunc) bool {
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

func newSignature(t sqlschema.Table, eq TypeEquivalenceFunc) signature {
	s := signature{
		underlying: make(map[sqlschema.ColumnDefinition]int),
		eq:         eq,
	}
	s.scan(t)
	return s
}

// scan iterates over table's field and counts occurrences of each unique column definition.
func (s *signature) scan(t sqlschema.Table) {
	for _, icol := range t.GetColumns() {
		scanCol := icol.(sqlschema.ColumnDefinition)
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

// refMap is a utility for tracking superficial changes in foreign keys,
// which do not require any modificiation in the database.
// Modern SQL dialects automatically updated foreign key constraints whenever
// a column or a table is renamed. Detector can use refMap to ignore any
// differences in foreign keys which were caused by renamed column/table.
type refMap map[*sqlschema.ForeignKey]string

func newRefMap(fks map[sqlschema.ForeignKey]string) refMap {
	rm := make(map[*sqlschema.ForeignKey]string)
	for fk, name := range fks {
		rm[&fk] = name
	}
	return rm
}

// RenameT updates table name in all foreign key definions which depend on it.
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

// RenameColumn updates column name in all foreign key definions which depend on it.
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

// Deref returns copies of ForeignKey values to a map. 
func (rm refMap) Deref() map[sqlschema.ForeignKey]string {
	out := make(map[sqlschema.ForeignKey]string)
	for fk, name := range rm {
		out[*fk] = name
	}
	return out
}
