package pgdialect

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"

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
)

func arrayAppend(fmter schema.Formatter, b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case int64:
		return strconv.AppendInt(b, v, 10)
	case float64:
		return dialect.AppendFloat64(b, v)
	case bool:
		return dialect.AppendBool(b, v)
	case []byte:
		return arrayAppendBytes(b, v)
	case string:
		return arrayAppendString(b, v)
	case time.Time:
		return fmter.Dialect().AppendTime(b, v)
	default:
		err := fmt.Errorf("pgdialect: can't append %T", v)
		return dialect.AppendError(b, err)
	}
}

func arrayAppendStringValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	return arrayAppendString(b, v.String())
}

func arrayAppendBytesValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	return arrayAppendBytes(b, v.Bytes())
}

func arrayAppendDriverValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	iface, err := v.Interface().(driver.Valuer).Value()
	if err != nil {
		return dialect.AppendError(b, err)
	}
	return arrayAppend(fmter, b, iface)
}

//------------------------------------------------------------------------------

func (d *Dialect) arrayAppender(typ reflect.Type) schema.AppenderFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := d.arrayAppender(typ.Elem()); fn != nil {
			return schema.PtrAppender(fn)
		}
	case reflect.Slice, reflect.Array:
		// ok:
	default:
		return nil
	}

	elemType := typ.Elem()

	if kind == reflect.Slice {
		switch elemType {
		case stringType:
			return appendStringSliceValue
		case intType:
			return appendIntSliceValue
		case int64Type:
			return appendInt64SliceValue
		case float64Type:
			return appendFloat64SliceValue
		}
	}

	appendElem := d.arrayElemAppender(elemType)
	if appendElem == nil {
		panic(fmt.Errorf("pgdialect: %s is not supported", typ))
	}

	return func(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
		kind := v.Kind()
		switch kind {
		case reflect.Ptr, reflect.Slice:
			if v.IsNil() {
				return dialect.AppendNull(b)
			}
		}

		if kind == reflect.Ptr {
			v = v.Elem()
		}

		b = append(b, '\'')

		b = append(b, '{')
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			b = appendElem(fmter, b, elem)
			b = append(b, ',')
		}
		if v.Len() > 0 {
			b[len(b)-1] = '}' // Replace trailing comma.
		} else {
			b = append(b, '}')
		}

		b = append(b, '\'')

		return b
	}
}

func (d *Dialect) arrayElemAppender(typ reflect.Type) schema.AppenderFunc {
	if typ.Implements(driverValuerType) {
		return arrayAppendDriverValue
	}
	switch typ.Kind() {
	case reflect.String:
		return arrayAppendStringValue
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return arrayAppendBytesValue
		}
	}
	return schema.Appender(d, typ)
}

func appendStringSliceValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	ss := v.Convert(sliceStringType).Interface().([]string)
	return appendStringSlice(b, ss)
}

func appendStringSlice(b []byte, ss []string) []byte {
	if ss == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, s := range ss {
		b = arrayAppendString(b, s)
		b = append(b, ',')
	}
	if len(ss) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func appendIntSliceValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	ints := v.Convert(sliceIntType).Interface().([]int)
	return appendIntSlice(b, ints)
}

func appendIntSlice(b []byte, ints []int) []byte {
	if ints == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, n := range ints {
		b = strconv.AppendInt(b, int64(n), 10)
		b = append(b, ',')
	}
	if len(ints) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func appendInt64SliceValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	ints := v.Convert(sliceInt64Type).Interface().([]int64)
	return appendInt64Slice(b, ints)
}

func appendInt64Slice(b []byte, ints []int64) []byte {
	if ints == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, n := range ints {
		b = strconv.AppendInt(b, n, 10)
		b = append(b, ',')
	}
	if len(ints) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func appendFloat64SliceValue(fmter schema.Formatter, b []byte, v reflect.Value) []byte {
	floats := v.Convert(sliceFloat64Type).Interface().([]float64)
	return appendFloat64Slice(b, floats)
}

func appendFloat64Slice(b []byte, floats []float64) []byte {
	if floats == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, n := range floats {
		b = dialect.AppendFloat64(b, n)
		b = append(b, ',')
	}
	if len(floats) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

//------------------------------------------------------------------------------

func arrayAppendBytes(b []byte, bs []byte) []byte {
	if bs == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, `"\\x`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

	b = append(b, '"')

	return b
}

func arrayAppendString(b []byte, s string) []byte {
	b = append(b, '"')
	for _, r := range s {
		switch r {
		case 0:
			// ignore
		case '\'':
			b = append(b, "''"...)
		case '"':
			b = append(b, '\\', '"')
		case '\\':
			b = append(b, '\\', '\\')
		default:
			if r < utf8.RuneSelf {
				b = append(b, byte(r))
				break
			}
			l := len(b)
			if cap(b)-l < utf8.UTFMax {
				b = append(b, make([]byte, utf8.UTFMax)...)
			}
			n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
			b = b[:l+n]
		}
	}
	b = append(b, '"')
	return b
}

//------------------------------------------------------------------------------

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
