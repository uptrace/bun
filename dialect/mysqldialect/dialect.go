package mysqldialect

import (
	"database/sql"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

const datetimeType = "DATETIME"

type Dialect struct {
	schema.BaseDialect

	tables   *schema.Tables
	features feature.Feature
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
	b = append(b, '\'')
	b = tm.AppendFormat(b, "2006-01-02 15:04:05.999999")
	b = append(b, '\'')
	return b
}

func (d *Dialect) AppendBytes(b []byte, bs []byte) []byte {
	if bs == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, `X'`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

	b = append(b, '\'')

	return b
}

func (d *Dialect) AppendJSON(b, jsonb []byte) []byte {
	b = append(b, '\'')

	for _, c := range jsonb {
		switch c {
		case '\'':
			b = append(b, "''"...)
		case '\\':
			b = append(b, `\\`...)
		default:
			b = append(b, c)
		}
	}

	b = append(b, '\'')

	return b
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
