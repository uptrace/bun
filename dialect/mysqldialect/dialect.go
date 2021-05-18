package mysqldialect

import (
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
	d.features = feature.Backticks |
		feature.AutoIncrement |
		feature.DefaultPlaceholder |
		feature.ValuesRow |
		feature.TableTruncate
	return d
}

func (d *Dialect) Name() string {
	return dialect.MySQL
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) OnTable(table *schema.Table) {
	for _, field := range table.Fields {
		field.DiscoveredSQLType = sqlType(field)
	}
}

func (d *Dialect) IdentQuote() byte {
	return '`'
}

func (d *Dialect) Append(fmter schema.Formatter, b []byte, v interface{}) []byte {
	return schema.Append(fmter, b, v)
}

func (d *Dialect) Appender(typ reflect.Type) schema.AppenderFunc {
	if v, ok := d.appenderMap.Load(typ); ok {
		return v.(schema.AppenderFunc)
	}

	fn := schema.Appender(typ)

	if v, ok := d.appenderMap.LoadOrStore(typ, fn); ok {
		return v.(schema.AppenderFunc)
	}
	return fn
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

func sqlType(field *schema.Field) string {
	switch field.DiscoveredSQLType {
	case sqltype.VarChar:
		return field.DiscoveredSQLType + "(255)"
	case sqltype.Timestamp:
		return "DATETIME"
	}
	return field.DiscoveredSQLType
}
