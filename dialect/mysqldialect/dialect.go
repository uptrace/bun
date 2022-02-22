package mysqldialect

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/mod/semver"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

const datetimeType = "DATETIME"

func init() {
	if Version() != bun.Version() {
		panic(fmt.Errorf("mysqldialect and Bun must have the same version: v%s != v%s",
			Version(), bun.Version()))
	}
}

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
		feature.TableNotExists |
		feature.InsertIgnore |
		feature.InsertOnDuplicateKey |
		feature.SelectExists
	return d
}

func (d *Dialect) Init(db *sql.DB) {
	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
		log.Printf("can't discover MySQL version: %s", err)
		return
	}

	if strings.Contains(version, "MariaDB") {
		version = semver.MajorMinor("v" + cleanupVersion(version))
		if semver.Compare(version, "v10.5.0") >= 0 {
			d.features |= feature.InsertReturning
		}
		return
	}

	version = semver.MajorMinor("v" + cleanupVersion(version))
	if semver.Compare(version, "v8.0") >= 0 {
		d.features |= feature.CTE | feature.WithValues | feature.DeleteTableAlias
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

func (*Dialect) AppendTime(b []byte, tm time.Time) []byte {
	b = append(b, '\'')
	b = tm.AppendFormat(b, "2006-01-02 15:04:05.999999")
	b = append(b, '\'')
	return b
}

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
