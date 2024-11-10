package sqlschema

import (
	"fmt"

	"github.com/uptrace/bun/schema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Table interface {
	GetSchema() string
	GetName() string
	GetColumns() *orderedmap.OrderedMap[string, Column]
	GetPrimaryKey() *PrimaryKey
	GetUniqueConstraints() []Unique
	GetFQN() FQN
}

var _ Table = (*BaseTable)(nil)

// BaseTable is a base table definition.
//
// Dialects and only dialects can use it to implement the Table interface.
// Other packages must use the Table interface.
type BaseTable struct {
	Schema string
	Name   string

	// ColumnDefinitions map each column name to the column definition.
	Columns *orderedmap.OrderedMap[string, Column]

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

func (td *BaseTable) GetColumns() *orderedmap.OrderedMap[string, Column] {
	return td.Columns
}

func (td *BaseTable) GetPrimaryKey() *PrimaryKey {
	return td.PrimaryKey
}

func (td *BaseTable) GetUniqueConstraints() []Unique {
	return td.UniqueConstraints
}

func (t *BaseTable) GetFQN() FQN {
	return FQN{Schema: t.Schema, Table: t.Name}
}

// FQN uniquely identifies a table in a multi-schema setup.
type FQN struct {
	Schema string
	Table  string
}

var _ schema.QueryAppender = (*FQN)(nil)

// AppendQuery appends a fully-qualified table name.
func (fqn *FQN) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return fmter.AppendQuery(b, "?.?", schema.Ident(fqn.Schema), schema.Ident(fqn.Table)), nil
}

func (fqn *FQN) String() string {
	return fmt.Sprintf("%s.%s", fqn.Schema, fqn.Table)
}
