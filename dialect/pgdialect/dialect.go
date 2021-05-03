package pgdialect

import (
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/sqlfmt"
)

type Dialect struct {
	tables   *schema.Tables
	features feature.Feature
}

func New() *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.Returning | feature.DropTableCascade
	return d
}

func (d *Dialect) Name() string {
	return dialect.PG
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) Append(fmter sqlfmt.QueryFormatter, b []byte, value interface{}) []byte {
	return sqlfmt.Append(fmter, b, value)
}

func (d *Dialect) OnField(field *schema.Field) {
	field.DiscoveredSQLType = fieldSQLType(field)

	switch field.DiscoveredSQLType {
	case sqltype.SmallInt:
		field.CreateTableSQLType = pgTypeSmallSerial
	case sqltype.Integer:
		field.CreateTableSQLType = pgTypeSerial
	case sqltype.BigInt:
		field.CreateTableSQLType = pgTypeBigSerial
	}

	if field.Tag.HasOption("array") {
		field.Append = arrayAppender(field.Type)
		field.Scan = arrayScanner(field.Type)
	}
}

func (d *Dialect) OnTable(table *schema.Table) {}

func (d *Dialect) Features() feature.Feature {
	return d.features
}
