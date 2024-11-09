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

// BaseTable is a base table definition.
// It MUST only be used by dialects to implement the Table interface.
type BaseTable struct {
	Schema string
	Name   string

	// ColumnDefinitions map each column name to the column definition.
	// TODO: this must be an ordered map so the order of columns is preserved
	Columns map[string]Column

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

func (td *BaseTable) GetSchema() string {
	return td.Schema
}

func (td *BaseTable) GetName() string {
	return td.Name
}

func (td *BaseTable) GetColumns() []Column {
	var columns []Column
	// FIXME: columns will be returned in a random order
	for colName := range td.Columns {
		columns = append(columns, td.Columns[colName])
	}
	return columns
}

func (td *BaseTable) GetPrimaryKey() *PrimaryKey {
	return td.PrimaryKey
}

func (td *BaseTable) GetUniqueConstraints() []Unique {
	return td.UniqueConstraints
}

func (t *BaseTable) GetFQN() schema.FQN {
	return schema.FQN{Schema: t.Schema, Table: t.Name}
}
