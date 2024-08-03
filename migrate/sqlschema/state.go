package sqlschema

import (
	"strings"

	"github.com/uptrace/bun/schema"
)

type State struct {
	Tables []Table
	FKs    map[FK]string
}

type Table struct {
	Schema  string
	Name    string
	Model   interface{}
	Columns map[string]Column
}

// T returns a fully-qualified name object for the table.
func (t *Table) T() tFQN {
	return T(t.Schema, t.Name)
}

// Column stores attributes of a database column.
type Column struct {
	SQLType         string
	DefaultValue    string
	IsPK            bool
	IsNullable      bool
	IsAutoIncrement bool
	IsIdentity      bool
}

// EqualSignatures determines if two tables have the same "signature".
func EqualSignatures(t1, t2 Table) bool {
	sig1 := newSignature(t1)
	sig2 := newSignature(t2)
	return sig1.Equals(sig2)
}

// signature is a set of column definitions, which allows "relation/name-agnostic" comparison between them;
// meaning that two columns are considered equal if their types are the same.
type signature struct {

	// underlying stores the number of occurences for each unique column type.
	// It helps to account for the fact that a table might have multiple columns that have the same type.
	underlying map[Column]int
}

func newSignature(t Table) signature {
	s := signature{
		underlying: make(map[Column]int),
	}
	s.scan(t)
	return s
}

// scan iterates over table's field and counts occurrences of each unique column definition.
func (s *signature) scan(t Table) {
	for _, c := range t.Columns {
		s.underlying[c]++
	}
}

// Equals returns true if 2 signatures share an identical set of columns.
func (s *signature) Equals(other signature) bool {
	if len(s.underlying) != len(other.underlying) {
		return false
	}
	for k, count := range s.underlying {
		if countOther, ok := other.underlying[k]; !ok || countOther != count {
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
	return cFQN{tFQN: T(schema, table), Column: newComposite(columns...)}
}

// T returns the FQN of the column's parent table.
func (c cFQN) T() tFQN {
	return c.tFQN
}

// composite is a hashable representation of []string used to define FKs that depend on multiple columns.
// Although having duplicated column references in a FK is illegal, composite neither validates nor enforces this constraint on the caller.
type composite string

// newComposite creates a composite column from a slice of column names.
func newComposite(columns ...string) composite {
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
func (c composite) Contains(other composite) bool {
	var count int
	checkColumns := other.Split()
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
			return newComposite(columns...)
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
	case fk.From.Column.Contains(c.Column):
		return true, &fk.From
	case fk.To.Column.Contains(c.Column):
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

// UpdateC updates the column FQN in all FKs that depend on it, e.g. if a column is renamed,
// and so, only the column-name part of the FQN can be updated. Returns the number of updated entries.
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
