package sqlitedialect

import (
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

type Dialect struct {
	tables   *schema.Tables
	features feature.Feature
}

func New() *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.Returning
	return d
}

func (d *Dialect) Name() string {
	return dialect.SQLite
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) OnField(field *schema.Field) {
	// INTEGER PRIMARY KEY is an alias for the ROWID.
	// It is safe to convert all ints to INTEGER, because SQLite types don't have size.
	switch field.DiscoveredSQLType {
	case sqltype.SmallInt, sqltype.BigInt:
		field.DiscoveredSQLType = sqltype.Integer
	}
}

func (d *Dialect) OnTable(table *schema.Table) {}
