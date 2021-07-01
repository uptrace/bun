package mysqldialect

import (
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

func appender(typ reflect.Type) schema.AppenderFunc {
	if typ == timeType {
		return appendTimeValue
	}
	return schema.Appender(typ)
}

func appendTimeValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	return appendTime(b, v.Interface().(time.Time))
}

func appendTime(b []byte, tm time.Time) []byte {
	if tm.IsZero() {
		return dialect.AppendNull(b)
	}
	b = append(b, '\'')
	b = tm.UTC().AppendFormat(b, "2006-01-02 15:04:05.999999")
	b = append(b, '\'')
	return b
}
