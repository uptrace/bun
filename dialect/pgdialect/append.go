package pgdialect

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var (
	driverValuerType = reflect.TypeOf((*driver.Valuer)(nil)).Elem()

	stringType      = reflect.TypeOf((*string)(nil)).Elem()
	sliceStringType = reflect.TypeOf([]string(nil))

	intType      = reflect.TypeOf((*int)(nil)).Elem()
	sliceIntType = reflect.TypeOf([]int(nil))

	int64Type      = reflect.TypeOf((*int64)(nil)).Elem()
	sliceInt64Type = reflect.TypeOf([]int64(nil))

	float64Type      = reflect.TypeOf((*float64)(nil)).Elem()
	sliceFloat64Type = reflect.TypeOf([]float64(nil))

	timeType      = reflect.TypeOf((*time.Time)(nil)).Elem()
	sliceTimeType = reflect.TypeOf([]time.Time(nil))
)

func appendTime(buf []byte, tm time.Time) []byte {
	return tm.UTC().AppendFormat(buf, "2006-01-02 15:04:05.999999-07:00")
}

var mapStringStringType = reflect.TypeOf(map[string]string(nil))

func (d *Dialect) hstoreAppender(typ reflect.Type) schema.AppenderFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := d.hstoreAppender(typ.Elem()); fn != nil {
			return schema.PtrAppender(fn)
		}
	case reflect.Map:
		// ok:
	default:
		return nil
	}

	if typ.Key() == stringType && typ.Elem() == stringType {
		return appendMapStringStringValue
	}

	return func(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
		err := fmt.Errorf("bun: Hstore(unsupported %s)", v.Type())
		return dialect.AppendError(b, err)
	}
}

func appendMapStringString(b []byte, m map[string]string) []byte {
	if m == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	for key, value := range m {
		b = arrayAppendString(b, key)
		b = append(b, '=', '>')
		b = arrayAppendString(b, value)
		b = append(b, ',')
	}
	if len(m) > 0 {
		b = b[:len(b)-1] // Strip trailing comma.
	}

	b = append(b, '\'')

	return b
}

func appendMapStringStringValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	m := v.Convert(mapStringStringType).Interface().(map[string]string)
	return appendMapStringString(b, m)
}
