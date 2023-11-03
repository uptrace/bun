package sqlschema

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type InspectorDialect interface {
	schema.Dialect
	Inspector(db *bun.DB, excludeTables ...string) Inspector
}

type Inspector interface {
	Inspect(ctx context.Context) (State, error)
}

type inspector struct {
	Inspector
}

func NewInspector(db *bun.DB, excludeTables ...string) (Inspector, error) {
	dialect, ok := (db.Dialect()).(InspectorDialect)
	if !ok {
		return nil, fmt.Errorf("%s does not implement sqlschema.Inspector", db.Dialect().Name())
	}
	return &inspector{
		Inspector: dialect.Inspector(db, excludeTables...),
	}, nil
}

// SchemaInspector creates the current project state from the passed bun.Models.
// Do not recycle SchemaInspector for different sets of models, as older models will not be de-registerred before the next run.
type SchemaInspector struct {
	tables *schema.Tables
}

var _ Inspector = (*SchemaInspector)(nil)

func NewSchemaInspector(tables *schema.Tables) *SchemaInspector {
	return &SchemaInspector{
		tables: tables,
	}
}

func (si *SchemaInspector) Inspect(ctx context.Context) (State, error) {
	var state State
	for _, t := range si.tables.All() {
		columns := make(map[string]Column)
		for _, f := range t.Fields {
			columns[f.Name] = Column{
				SQLType:         strings.ToLower(f.CreateTableSQLType),
				DefaultValue:    f.SQLDefault,
				IsPK:            f.IsPK,
				IsNullable:      !f.NotNull,
				IsAutoIncrement: f.AutoIncrement,
				IsIdentity:      f.Identity,
			}
		}

		state.Tables = append(state.Tables, Table{
			Schema:  t.Schema,
			Name:    t.Name,
			Model:   t.ZeroIface,
			Columns: columns,
		})
	}
	return state, nil
}
