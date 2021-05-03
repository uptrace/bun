package schema

import (
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/sqlfmt"
)

type Dialect interface {
	Name() string
	Tables() *Tables

	Append(fmter sqlfmt.QueryFormatter, b []byte, value interface{}) []byte
	OnField(field *Field)
	OnTable(table *Table)

	Features() feature.Feature
}
