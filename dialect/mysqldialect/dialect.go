package mysqldialect

import (
	"database/sql"
	"log"
	"reflect"
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

func (d *Dialect) Append(fmter schema.Formatter, b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case time.Time:
		return appendTime(b, v)
	default:
		return schema.Append(fmter, b, v, customAppender)
	}
}

func (d *Dialect) Appender(typ reflect.Type) schema.AppenderFunc {
	if v, ok := d.appenderMap.Load(typ); ok {
		return v.(schema.AppenderFunc)
	}

	fn := appender(typ)

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
