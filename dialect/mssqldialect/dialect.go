package mssqldialect

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

const (
	datetimeType  = "DATETIME"
	bitType       = "BIT"
	nvarcharType  = "NVARCHAR(MAX)"
	varbinaryType = "VARBINARY(MAX)"
)

type Dialect struct {
	schema.BaseDialect

	tables   *schema.Tables
	features feature.Feature
}

func New() *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.CTE |
		feature.DefaultPlaceholder |
		feature.Identity |
		feature.Output |
		feature.OffsetFetch
	return d
}

func (d *Dialect) Init(db *sql.DB) {
	var version string
	if err := db.QueryRow("SELECT @@VERSION").Scan(&version); err != nil {
		log.Printf("can't discover MSSQL version: %s", err)
		return
	}

	version = semver.MajorMinor("v" + cleanupVersion(version))
}

func cleanupVersion(v string) string {
	if s := strings.Index(v, " - "); s != -1 {
		if e := strings.Index(v[s+3:], " "); e != -1 {
			return v[s+3 : s+3+e]
		}
	}
	return ""
}

func (d *Dialect) Name() dialect.Name {
	return dialect.MSSQL
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
		if strings.ToUpper(field.UserSQLType) == sqltype.JSON {
			field.UserSQLType = nvarcharType
		}
	}
}

func (d *Dialect) IdentQuote() byte {
	return '"'
}

func (*Dialect) AppendTime(b []byte, tm time.Time) []byte {
	b = append(b, '\'')
	b = tm.AppendFormat(b, "2006-01-02 15:04:05.999999")
	b = append(b, '\'')
	return b
}

func (*Dialect) AppendBytes(b, bs []byte) []byte {
	if bs == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, "0x"...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

	return b
}

func sqlType(field *schema.Field) string {
	switch field.DiscoveredSQLType {
	case sqltype.VarChar:
		return field.DiscoveredSQLType + "(255)"
	case sqltype.Timestamp:
		return datetimeType
	case sqltype.Boolean:
		return bitType
	case sqltype.JSON:
		return nvarcharType
	case sqltype.Blob:
		return varbinaryType
	}
	return field.DiscoveredSQLType
}