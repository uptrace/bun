package pgdialect

import (
	"database/sql"
	"reflect"
	"strconv"
	"sync"
	"time"

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
	d.features = feature.CTE |
		feature.Returning |
		feature.DefaultPlaceholder |
		feature.DoubleColonCast |
		feature.InsertTableAlias |
		feature.DeleteTableAlias |
		feature.TableCascade |
		feature.TableIdentity |
		feature.TableTruncate
	return d
}

func (d *Dialect) Init(*sql.DB) {}

func (d *Dialect) Name() dialect.Name {
	return dialect.PG
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
	field.DiscoveredSQLType = fieldSQLType(field)

	if field.AutoIncrement {
		switch field.DiscoveredSQLType {
		case sqltype.SmallInt:
			field.CreateTableSQLType = pgTypeSmallSerial
		case sqltype.Integer:
			field.CreateTableSQLType = pgTypeSerial
		case sqltype.BigInt:
			field.CreateTableSQLType = pgTypeBigSerial
		}
	}

	if field.Tag.HasOption("array") {
		field.Append = arrayAppender(field.StructField.Type)
		field.Scan = arrayScanner(field.StructField.Type)
	}
}

func (d *Dialect) IdentQuote() byte {
	return '"'
}

func (d *Dialect) Append(fmter schema.Formatter, b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case nil:
		return dialect.AppendNull(b)
	case bool:
		return dialect.AppendBool(b, v)
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, v, 10)
	case uint:
		return strconv.AppendInt(b, int64(v), 10)
	case uint32:
		return strconv.AppendInt(b, int64(v), 10)
	case uint64:
		return strconv.AppendInt(b, int64(v), 10)
	case float32:
		return dialect.AppendFloat32(b, v)
	case float64:
		return dialect.AppendFloat64(b, v)
	case string:
		return dialect.AppendString(b, v)
	case time.Time:
		return dialect.AppendTime(b, v)
	case []byte:
		return dialect.AppendBytes(b, v)
	case schema.QueryAppender:
		return schema.AppendQueryAppender(fmter, b, v)
	default:
		vv := reflect.ValueOf(v)
		if vv.Kind() == reflect.Ptr && vv.IsNil() {
			return dialect.AppendNull(b)
		}
		appender := d.Appender(vv.Type())
		return appender(fmter, b, vv)
	}
}

func (d *Dialect) Appender(typ reflect.Type) schema.AppenderFunc {
	if v, ok := d.appenderMap.Load(typ); ok {
		return v.(schema.AppenderFunc)
	}

	fn := schema.Appender(typ, customAppender)

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
