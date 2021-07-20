package sqlitedialect

import (
	"database/sql"
	"reflect"
	"sync"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

type Dialect struct {
	tables   *schema.Tables
	features feature.Feature

	appenderMap sync.Map
	scannerMap  sync.Map
}

func New() *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.Returning | feature.InsertTableAlias | feature.DeleteTableAlias
	return d
}

func (d *Dialect) Init(*sql.DB) {}

func (d *Dialect) Name() dialect.Name {
	return dialect.SQLite
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) OnTable(table *schema.Table) {
	for _, field := range table.FieldMap {
		d.onField(field)
	}
}

func (d *Dialect) onField(field *schema.Field) {
	// INTEGER PRIMARY KEY is an alias for the ROWID.
	// It is safe to convert all ints to INTEGER, because SQLite types don't have size.
	switch field.DiscoveredSQLType {
	case sqltype.SmallInt, sqltype.BigInt:
		field.DiscoveredSQLType = sqltype.Integer
	}
}

func (d *Dialect) IdentQuote() byte {
	return '"'
}

func (d *Dialect) Append(fmter schema.Formatter, b []byte, v interface{}) []byte {
	return schema.Append(fmter, b, v, nil)
}

func (d *Dialect) Appender(typ reflect.Type) schema.AppenderFunc {
	if v, ok := d.appenderMap.Load(typ); ok {
		return v.(schema.AppenderFunc)
	}

	fn := schema.Appender(typ, nil)

	if v, ok := d.appenderMap.LoadOrStore(typ, fn); ok {
		return v.(schema.AppenderFunc)
	}
	return fn
}

func (d *Dialect) FieldAppender(field *schema.Field) schema.AppenderFunc {
	return schema.FieldAppender(d, field)
}

func (d *Dialect) Scanner(typ reflect.Type) schema.ScannerFunc {
	if v, ok := d.scannerMap.Load(typ); ok {
		return v.(schema.ScannerFunc)
	}

	fn := scanner(typ)

	if v, ok := d.scannerMap.LoadOrStore(typ, fn); ok {
		return v.(schema.ScannerFunc)
	}
	return fn
}
