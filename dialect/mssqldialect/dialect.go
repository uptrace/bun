package mssqldialect

import (
	"database/sql"
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
const bitType = "BIT"

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
		feature.Identity
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

/*
func (*Dialect) AppendString(b []byte, s string) []byte {
	b = append(b, '\'')
loop:
	for _, r := range s {
		switch r {
		case '\000':
			continue loop
		case '\'':
			b = append(b, "''"...)
			continue loop
		case '\\':
			b = append(b, '\\', '\\')
			continue loop
		}

		if r < utf8.RuneSelf {
			b = append(b, byte(r))
			continue
		}

		l := len(b)
		if cap(b)-l < utf8.UTFMax {
			b = append(b, make([]byte, utf8.UTFMax)...)
		}
		n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
		b = b[:l+n]
	}
	b = append(b, '\'')
	return b
}

func (*Dialect) AppendBytes(b []byte, bs []byte) []byte {
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

func (*Dialect) AppendJSON(b, jsonb []byte) []byte {
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
}*/

func sqlType(field *schema.Field) string {
	switch field.DiscoveredSQLType {
	case sqltype.VarChar:
		return field.DiscoveredSQLType + "(255)"
	case sqltype.Timestamp:
		return datetimeType
	case sqltype.Boolean:
		return bitType
	}
	return field.DiscoveredSQLType
}
