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
	state := State{
		FKs: make(map[FK]string),
	}
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

		for _, rel := range t.Relations {
			// These relations are nominal and do not need a foreign key to be declared in the current table.
			// They will be either expressed as N:1 relations in an m2m mapping table, or will be referenced by the other table if it's a 1:N.
			if rel.Type == schema.ManyToManyRelation ||
				rel.Type == schema.HasManyRelation {
				continue
			}

			var fromCols, toCols []string
			for _, f := range rel.BaseFields {
				fromCols = append(fromCols, f.Name)
			}
			for _, f := range rel.JoinFields {
				toCols = append(toCols, f.Name)
			}

			target := rel.JoinTable
			state.FKs[FK{
				From: C(t.Schema, t.Name, fromCols...),
				To:   C(target.Schema, target.Name, toCols...),
			}] = ""
		}
	}
	return state, nil
}
