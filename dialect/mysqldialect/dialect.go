package mysqldialect

import (
	"database/sql"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

const datetimeType = "DATETIME"

type Dialect struct {
	tables   *schema.Tables
	features feature.Feature

	appenderMap sync.Map
	scannerMap  sync.Map
}

func New() *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.AutoIncrement |
		feature.DefaultPlaceholder |
		feature.UpdateMultiTable |
		feature.ValuesRow |
		feature.TableTruncate |
		feature.OnDuplicateKey
	return d
}

func (d *Dialect) Init(db *sql.DB) {
	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
		log.Printf("can't discover MySQL version: %s", err)
		return
	}

	if strings.Contains(version, "MariaDB") {
		return
	}

	version = semver.MajorMinor("v" + cleanupVersion(version))
	if semver.Compare(version, "v8.0") >= 0 {
		d.features |= feature.CTE | feature.DeleteTableAlias
	}
}

func cleanupVersion(s string) string {
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return s[:i]
	}
	return s
}

func (d *Dialect) Name() dialect.Name {
	return dialect.MySQL
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) OnTable(table *schema.Table) {
	for _, field := range table.FieldMap {
		field.DiscoveredSQLType = sqlType(field)
	}
}

func (d *Dialect) IdentQuote() byte {
	return '`'
}

func (d *Dialect) AppendTime(b []byte, tm time.Time) []byte {
	return appendTime(b, tm)
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
		return strconv.AppendUint(b, uint64(v), 10)
	case uint32:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint64:
		return strconv.AppendUint(b, v, 10)
	case float32:
		return dialect.AppendFloat32(b, v)
	case float64:
		return dialect.AppendFloat64(b, v)
	case string:
		return dialect.AppendString(b, v)
	case time.Time:
		return appendTime(b, v)
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
	switch strings.ToUpper(field.UserSQLType) {
	case sqltype.JSON:
		return appendJSONValue
	}

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

func sqlType(field *schema.Field) string {
	switch field.DiscoveredSQLType {
	case sqltype.VarChar:
		return field.DiscoveredSQLType + "(255)"
	case sqltype.Timestamp:
		return datetimeType
	}
	return field.DiscoveredSQLType
}
