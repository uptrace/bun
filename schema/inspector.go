package schema

import (
	"context"
	"strings"
)

type Inspector interface {
	Inspect(ctx context.Context) (State, error)
}

type State struct {
	Tables []TableDef
}

type TableDef struct {
	Schema string
	Name string
	Columns map[string]ColumnDef
}

type ColumnDef struct {
	SQLType         string
	DefaultValue    string
	IsPK            bool
	IsNullable      bool
	IsAutoIncrement bool
	IsIdentity      bool
}

type SchemaInspector struct {
	dialect Dialect
}

var _ Inspector = (*SchemaInspector)(nil)

func NewInspector(dialect Dialect, models ...interface{}) *SchemaInspector {
	dialect.Tables().Register(models...)
	return &SchemaInspector{
		dialect: dialect,
	}
}

func (si *SchemaInspector) Inspect(ctx context.Context) (State, error) {
	var state State
	for _, t := range si.dialect.Tables().All() {
		columns := make(map[string]ColumnDef)
		for _, f := range t.Fields {
			columns[f.Name] = ColumnDef{
				SQLType: f.CreateTableSQLType,
				DefaultValue: f.SQLDefault,
				IsPK: f.IsPK,
				IsNullable: !f.NotNull,
				IsAutoIncrement: f.AutoIncrement,
				IsIdentity: f.Identity,
			}
		}

		schema, table := splitTableNameTag(si.dialect, t.Name)
		state.Tables = append(state.Tables, TableDef{
			Schema: schema,
			Name: table,
			Columns: columns,
		})
	}
	return state, nil
}

// splitTableNameTag
func splitTableNameTag(d Dialect, nameTag string) (string, string) {
	schema, table := d.DefaultSchema(), nameTag
	if schemaTable := strings.Split(nameTag, "."); len(schemaTable) == 2 {
		schema, table = schemaTable[0], schemaTable[1]
	}
	return schema, table
}