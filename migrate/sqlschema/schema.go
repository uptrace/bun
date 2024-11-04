package sqlschema

import (
	"fmt"
	"slices"
	"strings"

	"github.com/uptrace/bun/schema"
)

type DatabaseSchema struct {
	TableDefinitions map[string]TableDefinition
	ForeignKeys      map[ForeignKey]string
}

type TableDefinition struct {
	Schema string
	Name   string

	// ColumnDefimitions map each column name to the column definition.
	ColumnDefimitions map[string]ColumnDefinition

	// PrimaryKey holds the primary key definition.
	// A nil value means that no primary key is defined for the table.
	PrimaryKey *PrimaryKey

	// UniqueConstraints defined on the table.
	UniqueContraints []Unique

	// Additional metadata that Inspector implementations might provide about the table.
	Additional interface{}
}

func (t TableDefinition) FQN() schema.FQN {
	return schema.FQN{Schema: t.Schema, Table: t.Name}
}

// ColumnDefinition stores attributes of a database column.
type ColumnDefinition struct {
	SQLType         string
	VarcharLen      int
	DefaultValue    string
	IsNullable      bool
	IsAutoIncrement bool
	IsIdentity      bool

	// Additional metadata that Inspector implementations might provide about the column.
	Additional interface{}

	// TODO: add Precision and Cardinality for timestamps/bit-strings/floats and arrays respectively.
}

// AppendQuery appends full SQL data type.
func (c *ColumnDefinition) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, c.SQLType...)
	if c.VarcharLen == 0 {
		return b, nil
	}
	b = append(b, "("...)
	b = append(b, fmt.Sprint(c.VarcharLen)...)
	b = append(b, ")"...)
	return b, nil
}

type ForeignKey struct {
	From ColumnReference
	To   ColumnReference
}

func NewColumnReference(schemaName, tableName string, columns ...string) ColumnReference {
	return ColumnReference{
		FQN:    schema.FQN{Schema: schemaName, Table: tableName},
		Column: NewColumns(columns...),
	}
}

func (fk ForeignKey) DependsOnTable(fqn schema.FQN) bool {
	return fk.From.FQN == fqn || fk.To.FQN == fqn
}

func (fk ForeignKey) DependsOnColumn(fqn schema.FQN, column string) bool {
	return fk.DependsOnTable(fqn) &&
		(fk.From.Column.Contains(column) || fk.To.Column.Contains(column))
}

// Columns is a hashable representation of []string used to define schema constraints that depend on multiple columns.
// Although having duplicated column references in these constraints is illegal, Columns neither validates nor enforces this constraint on the caller.
type Columns string

// NewColumns creates a composite column from a slice of column names.
func NewColumns(columns ...string) Columns {
	slices.Sort(columns)
	return Columns(strings.Join(columns, ","))
}

func (c *Columns) String() string {
	return string(*c)
}

func (c *Columns) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return schema.Safe(*c).AppendQuery(fmter, b)
}

// Split returns a slice of column names that make up the composite.
func (c *Columns) Split() []string {
	return strings.Split(c.String(), ",")
}

// ContainsColumns checks that columns in "other" are a subset of current colums.
func (c *Columns) ContainsColumns(other Columns) bool {
	columns := c.Split()
Outer:
	for _, check := range other.Split() {
		for _, column := range columns {
			if check == column {
				continue Outer
			}
		}
		return false
	}
	return true
}

// Contains checks that a composite column contains the current column.
func (c *Columns) Contains(other string) bool {
	return c.ContainsColumns(Columns(other))
}

// Replace renames a column if it is part of the composite.
// If a composite consists of multiple columns, only one column will be renamed.
func (c *Columns) Replace(oldColumn, newColumn string) bool {
	columns := c.Split()
	for i, column := range columns {
		if column == oldColumn {
			columns[i] = newColumn
			*c = NewColumns(columns...)
			return true
		}
	}
	return false
}

// Unique represents a unique constraint defined on 1 or more columns.
type Unique struct {
	Name    string
	Columns Columns
}

// Equals checks that two unique constraint are the same, assuming both are defined for the same table.
func (u Unique) Equals(other Unique) bool {
	return u.Columns == other.Columns
}

// PrimaryKey represents a primary key constraint defined on 1 or more columns.
type PrimaryKey struct {
	Name    string
	Columns Columns
}

type ColumnReference struct {
	FQN    schema.FQN
	Column Columns
}
