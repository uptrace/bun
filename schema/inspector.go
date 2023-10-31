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
	Schema  string
	Name    string
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
	tables *Tables
}

var _ Inspector = (*SchemaInspector)(nil)

func NewInspector(tables *Tables) *SchemaInspector {
	return &SchemaInspector{
		tables: tables,
	}
}

// Inspect creates the current project state from the passed bun.Models.
// Do not recycle SchemaInspector for different sets of models, as older models will not be de-registerred before the next run.
func (si *SchemaInspector) Inspect(ctx context.Context) (State, error) {
	var state State
	for _, t := range si.tables.All() {
		columns := make(map[string]ColumnDef)
		for _, f := range t.Fields {
			columns[f.Name] = ColumnDef{
				SQLType:         strings.ToLower(f.CreateTableSQLType),
				DefaultValue:    f.SQLDefault,
				IsPK:            f.IsPK,
				IsNullable:      !f.NotNull,
				IsAutoIncrement: f.AutoIncrement,
				IsIdentity:      f.Identity,
			}
		}

		state.Tables = append(state.Tables, TableDef{
			Schema:  t.Schema,
			Name:    t.Name,
			Columns: columns,
		})
	}
	return state, nil
}
