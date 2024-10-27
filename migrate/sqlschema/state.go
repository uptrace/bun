package sqlschema

import (
	"fmt"
	"slices"
	"strings"

	"github.com/uptrace/bun/schema"
)

type State struct {
	Tables []Table
	FKs    map[FK]string
}

type Table struct {
	// Schema containing the table.
	Schema string

	// Table name.
	Name string

	// Model stores a pointer to the bun's underlying Go struct for the table.
	Model interface{}

	// Columns map each column name to the column type definition.
	Columns map[string]Column

	// UniqueConstraints defined on the table.
	UniqueContraints []Unique

	// PrimaryKey holds the primary key definition if any.
	PK *PK
}

// T returns a fully-qualified name object for the table.
func (t *Table) T() tFQN {
	return T(t.Schema, t.Name)
}

// Column stores attributes of a database column.
type Column struct {
	SQLType         string
	VarcharLen      int
	DefaultValue    string
	IsNullable      bool
	IsAutoIncrement bool
	IsIdentity      bool
	// TODO: add Precision and Cardinality for timestamps/bit-strings/floats and arrays respectively.
}

// AppendQuery appends full SQL data type.
func (c *Column) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, c.SQLType...)
	if c.VarcharLen == 0 {
		return b, nil
	}
	b = append(b, "("...)
	b = append(b, fmt.Sprint(c.VarcharLen)...)
	b = append(b, ")"...)
	return b, nil
}

type TypeEquivalenceFunc func(Column, Column) bool

// EqualSignatures determines if two tables have the same "signature".
func EqualSignatures(t1, t2 Table, eq TypeEquivalenceFunc) bool {
	sig1 := newSignature(t1, eq)
	sig2 := newSignature(t2, eq)
	return sig1.Equals(sig2)
}

// signature is a set of column definitions, which allows "relation/name-agnostic" comparison between them;
// meaning that two columns are considered equal if their types are the same.
type signature struct {

	// underlying stores the number of occurences for each unique column type.
	// It helps to account for the fact that a table might have multiple columns that have the same type.
	underlying map[Column]int

	eq TypeEquivalenceFunc
}

func newSignature(t Table, eq TypeEquivalenceFunc) signature {
	s := signature{
		underlying: make(map[Column]int),
		eq:         eq,
	}
	s.scan(t)
	return s
}

// scan iterates over table's field and counts occurrences of each unique column definition.
func (s *signature) scan(t Table) {
	for _, scanCol := range t.Columns {
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
func (s *signature) getCount(keyCol Column) (key Column, count int) {
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

// tFQN is a fully-qualified table name.
type tFQN struct {
	Schema string
	Table  string
}

// T creates a fully-qualified table name object.
func T(schema, table string) tFQN { return tFQN{Schema: schema, Table: table} }

// cFQN is a fully-qualified column name.
type cFQN struct {
	tFQN
	Column composite
}

// C creates a fully-qualified column name object.
func C(schema, table string, columns ...string) cFQN {
	return cFQN{tFQN: T(schema, table), Column: NewComposite(columns...)}
}

// T returns the FQN of the column's parent table.
func (c cFQN) T() tFQN {
	return c.tFQN
}

// composite is a hashable representation of []string used to define FKs that depend on multiple columns.
// Although having duplicated column references in a FK is illegal, composite neither validates nor enforces this constraint on the caller.
type composite string

// NewComposite creates a composite column from a slice of column names.
func NewComposite(columns ...string) composite {
	slices.Sort(columns)
	return composite(strings.Join(columns, ","))
}

func (c composite) String() string {
	return string(c)
}

func (c composite) Safe() schema.Safe {
	return schema.Safe(c)
}

// Split returns a slice of column names that make up the composite.
func (c composite) Split() []string {
	return strings.Split(c.String(), ",")
}

// Contains checks that a composite column contains every part of another composite.
func (c composite) contains(other composite) bool {
	return c.Contains(string(other))
}

// Contains checks that a composite column contains the current column.
func (c composite) Contains(other string) bool {
	var count int
	checkColumns := composite(other).Split()
	wantCount := len(checkColumns)

	for _, check := range checkColumns {
		for _, column := range c.Split() {
			if check == column {
				count++
			}
			if count == wantCount {
				return true
			}
		}
	}
	return count == wantCount
}

// Replace renames a column if it is part of the composite.
// If a composite consists of multiple columns, only one column will be renamed.
func (c composite) Replace(oldColumn, newColumn string) composite {
	columns := c.Split()
	for i, column := range columns {
		if column == oldColumn {
			columns[i] = newColumn
			return NewComposite(columns...)
		}
	}
	return c
}

// FK defines a foreign key constraint.
//
// Example:
//
//	fk := FK{
//		From: C("a", "b", "c_1", "c_2"), // supports multicolumn FKs
//		To:	C("w", "x", "y_1", "y_2")
//	}
type FK struct {
	From cFQN // From is the referencing column.
	To   cFQN // To is the referenced column.
}

// dependsT checks if either part of the FK's definition mentions T
// and returns the columns that belong to T. Notice that *C allows modifying the column's FQN.
//
// Example:
//
//	FK{
//		From: C("a", "b", "c"),
//		To:	  C("x", "y", "z"),
//	}
//	depends on T("a", "b") and T("x", "y")
func (fk *FK) dependsT(t tFQN) (ok bool, cols []*cFQN) {
	if c := &fk.From; c.T() == t {
		ok = true
		cols = append(cols, c)
	}
	if c := &fk.To; c.T() == t {
		ok = true
		cols = append(cols, c)
	}
	if !ok {
		return false, nil
	}
	return
}

// dependsC checks if the FK definition mentions C and returns a modifiable FQN of the matching column.
//
// Example:
//
//	FK{
//		From: C("a", "b", "c_1", "c_2"),
//		To:	  C("w", "x", "y_1", "y_2"),
//	}
//	depends on C("a", "b", "c_1"), C("a", "b", "c_2"), C("w", "x", "y_1"), and C("w", "x", "y_2")
func (fk *FK) dependsC(c cFQN) (bool, *cFQN) {
	switch {
	case fk.From.Column.contains(c.Column):
		return true, &fk.From
	case fk.To.Column.contains(c.Column):
		return true, &fk.To
	}
	return false, nil
}

// RefMap helps detecting modified FK relations.
// It starts with an initial state and provides methods to update and delete
// foreign key relations based on the column or table they depend on.
//
// Note: this is only important/necessary if we want to rename FKs instead of re-creating them.
// Most of the time it wouldn't make a difference, but there may be cases in which re-creating FKs could be costly
// and renaming them would be preferred.
type RefMap map[FK]*FK

// deleted is a special value that RefMap uses to denote a deleted FK constraint.
var deleted FK

// NewRefMap records the FK's initial state to a RefMap.
func NewRefMap(fks ...FK) RefMap {
	ref := make(RefMap)
	for _, fk := range fks {
		copyfk := fk
		ref[fk] = &copyfk
	}
	return ref
}

// UpdateT updates the table FQN in all FKs that depend on it, e.g. if a table is renamed or moved to a different schema.
// Returns the number of updated entries.
func (r RefMap) UpdateT(oldT, newT tFQN) (n int) {
	for _, fk := range r {
		ok, cols := fk.dependsT(oldT)
		if !ok {
			continue
		}
		for _, c := range cols {
			c.Schema = newT.Schema
			c.Table = newT.Table
		}
		n++
	}
	return
}

// UpdateC updates the column FQN in all FKs that depend on it. E.g. if a column was renamed,
// only the column-name part of the FQN needs to be updated. Returns the number of updated entries.
func (r RefMap) UpdateC(oldC cFQN, newColumn string) (n int) {
	for _, fk := range r {
		if ok, col := fk.dependsC(oldC); ok {
			oldColumns := oldC.Column.Split()
			// updateC will only update 1 column per invocation.
			col.Column = col.Column.Replace(oldColumns[0], newColumn)
			n++
		}
	}
	return
}

// DeleteT marks all FKs that depend on the table as deleted.
// Returns the number of deleted entries.
func (r RefMap) DeleteT(t tFQN) (n int) {
	for old, fk := range r {
		if ok, _ := fk.dependsT(t); ok {
			r[old] = &deleted
			n++
		}
	}
	return
}

// DeleteC marks all FKs that depend on the column as deleted.
// Returns the number of deleted entries.
func (r RefMap) DeleteC(c cFQN) (n int) {
	for old, fk := range r {
		if ok, _ := fk.dependsC(c); ok {
			r[old] = &deleted
			n++
		}
	}
	return
}

// Updated returns FKs that were updated, both their old and new defitions.
func (r RefMap) Updated() map[FK]FK {
	fks := make(map[FK]FK)
	for old, fk := range r {
		if old != *fk {
			fks[old] = *fk
		}
	}
	return fks
}

// Deleted gets all FKs that were marked as deleted.
func (r RefMap) Deleted() (fks []FK) {
	for old, fk := range r {
		if fk == &deleted {
			fks = append(fks, old)
		}
	}
	return
}

// Unique represents a unique constraint defined on 1 or more columns.
type Unique struct {
	Name    string
	Columns composite
}

// Equals checks that two unique constraint are the same, assuming both are defined for the same table.
func (u Unique) Equals(other Unique) bool {
	return u.Columns == other.Columns
}

// PK represents a primary key constraint defined on 1 or more columns.
type PK struct {
	Name    string
	Columns composite
}
