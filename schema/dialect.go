package schema

import (
	"reflect"

	"github.com/uptrace/bun/dialect/feature"
)

type Dialect interface {
	Name() string
	Features() feature.Feature

	Tables() *Tables

	OnField(field *Field)
	OnTable(table *Table)

	IdentQuote() byte
	Appender(typ reflect.Type) AppenderFunc
}
