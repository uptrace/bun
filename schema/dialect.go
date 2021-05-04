package schema

import (
	"github.com/uptrace/bun/dialect/feature"
)

type Dialect interface {
	Name() string
	Features() feature.Feature

	Tables() *Tables

	OnField(field *Field)
	OnTable(table *Table)
}
