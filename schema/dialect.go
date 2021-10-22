package schema

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal/parser"
)

type Dialect interface {
	Init(db *sql.DB)

	Name() dialect.Name
	Features() feature.Feature

	Tables() *Tables
	OnTable(table *Table)

	IdentQuote() byte

	AppendUint32(b []byte, n uint32) []byte
	AppendUint64(b []byte, n uint64) []byte
	AppendTime(b []byte, tm time.Time) []byte
	AppendBytes(b []byte, bs []byte) []byte
	AppendJSON(b, jsonb []byte) []byte
}

//------------------------------------------------------------------------------

type BaseDialect struct{}

func (BaseDialect) AppendUint32(b []byte, n uint32) []byte {
	return strconv.AppendUint(b, uint64(n), 10)
}

func (BaseDialect) AppendUint64(b []byte, n uint64) []byte {
	return strconv.AppendUint(b, n, 10)
}

func (BaseDialect) AppendTime(b []byte, tm time.Time) []byte {
	b = append(b, '\'')
	b = tm.UTC().AppendFormat(b, "2006-01-02 15:04:05.999999-07:00")
	b = append(b, '\'')
	return b
}

func (BaseDialect) AppendBytes(b, bs []byte) []byte {
	return dialect.AppendBytes(b, bs)
}

func (BaseDialect) AppendJSON(b, jsonb []byte) []byte {
	b = append(b, '\'')

	p := parser.New(jsonb)
	for p.Valid() {
		c := p.Read()
		switch c {
		case '"':
			b = append(b, '"')
		case '\'':
			b = append(b, "''"...)
		case '\000':
			continue
		case '\\':
			if p.SkipBytes([]byte("u0000")) {
				b = append(b, `\\u0000`...)
			} else {
				b = append(b, '\\')
				if p.Valid() {
					b = append(b, p.Read())
				}
			}
		default:
			b = append(b, c)
		}
	}

	b = append(b, '\'')

	return b
}

//------------------------------------------------------------------------------

type nopDialect struct {
	BaseDialect

	tables   *Tables
	features feature.Feature
}

func newNopDialect() *nopDialect {
	d := new(nopDialect)
	d.tables = NewTables(d)
	d.features = feature.Returning
	return d
}

func (d *nopDialect) Init(*sql.DB) {}

func (d *nopDialect) Name() dialect.Name {
	return dialect.Invalid
}

func (d *nopDialect) Features() feature.Feature {
	return d.features
}

func (d *nopDialect) Tables() *Tables {
	return d.tables
}

func (d *nopDialect) OnField(field *Field) {}

func (d *nopDialect) OnTable(table *Table) {}

func (d *nopDialect) IdentQuote() byte {
	return '"'
}
