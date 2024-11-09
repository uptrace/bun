package sqlschema

import "github.com/uptrace/bun/schema"

type Table interface {
	GetSchema() string
	GetName() string
	GetColumns() []Column
	GetPrimaryKey() *PrimaryKey
	GetUniqueConstraints() []Unique
	GetFQN() schema.FQN
}

var _ Table = (*BaseTable)(nil)

type BaseTable struct {
	Schema string
	Name   string

	// ColumnDefinitions map each column name to the column definition.
	ColumnDefinitions map[string]*BaseColumn

	// PrimaryKey holds the primary key definition.
	// A nil value means that no primary key is defined for the table.
	PrimaryKey *PrimaryKey

	// UniqueConstraints defined on the table.
	UniqueConstraints []Unique
}

// PrimaryKey represents a primary key constraint defined on 1 or more columns.
type PrimaryKey struct {
	Name    string
	Columns Columns
}
