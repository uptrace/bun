package mysqldialect

import (
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/extra/bunjson"
	"github.com/uptrace/bun/schema"
)

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

func appender(typ reflect.Type) schema.AppenderFunc {
	if typ == timeType {
		return appendTimeValue
	}
	return schema.Appender(typ, customAppender)
}

func customAppender(typ reflect.Type) schema.AppenderFunc {
	switch typ.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
		return appendJSONValue
	}
	return nil
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

func appendJSONValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	bb, err := bunjson.Marshal(v.Interface())
	if err != nil {
		return dialect.AppendError(b, err)
	}

	if len(bb) > 0 && bb[len(bb)-1] == '\n' {
		bb = bb[:len(bb)-1]
	}

	return appendJSON(b, bb)
}

func appendJSON(b, jsonb []byte) []byte {
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
