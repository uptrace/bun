package schema

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/internal"
)

var (
	timeType           = reflect.TypeOf((*time.Time)(nil)).Elem()
	ipType             = reflect.TypeOf((*net.IP)(nil)).Elem()
	ipNetType          = reflect.TypeOf((*net.IPNet)(nil)).Elem()
	jsonRawMessageType = reflect.TypeOf((*json.RawMessage)(nil)).Elem()

	driverValuerType  = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	queryAppenderType = reflect.TypeOf((*QueryAppender)(nil)).Elem()
)

type AppenderFunc func(fmter Formatter, b []byte, v reflect.Value) []byte

var (
	appenders   []AppenderFunc
	appenderMap sync.Map
)

//nolint
func init() {
	appenders = []AppenderFunc{
		reflect.Bool:          appendBoolValue,
		reflect.Int:           AppendIntValue,
		reflect.Int8:          AppendIntValue,
		reflect.Int16:         AppendIntValue,
		reflect.Int32:         AppendIntValue,
		reflect.Int64:         AppendIntValue,
		reflect.Uint:          AppendUintValue,
		reflect.Uint8:         AppendUintValue,
		reflect.Uint16:        AppendUintValue,
		reflect.Uint32:        AppendUintValue,
		reflect.Uint64:        AppendUintValue,
		reflect.Uintptr:       nil,
		reflect.Float32:       appendFloat32Value,
		reflect.Float64:       appendFloat64Value,
		reflect.Complex64:     nil,
		reflect.Complex128:    nil,
		reflect.Array:         appendJSONValue,
		reflect.Chan:          nil,
		reflect.Func:          nil,
		reflect.Interface:     appendIfaceValue,
		reflect.Map:           appendJSONValue,
		reflect.Ptr:           nil,
		reflect.Slice:         appendJSONValue,
		reflect.String:        appendStringValue,
		reflect.Struct:        appendStructValue,
		reflect.UnsafePointer: nil,
	}
}

func Appender(typ reflect.Type) AppenderFunc {
	if v, ok := appenderMap.Load(typ); ok {
		return v.(AppenderFunc)
	}
	fn := appender(typ)
	_, _ = appenderMap.LoadOrStore(typ, fn)
	return fn
}

func appender(typ reflect.Type) AppenderFunc {
	switch typ {
	case timeType:
		return appendTimeValue
	case ipType:
		return appendIPValue
	case ipNetType:
		return appendIPNetValue
	case jsonRawMessageType:
		return appendJSONRawMessageValue
	}

	if typ.Implements(queryAppenderType) {
		return appendQueryAppenderValue
	}
	if typ.Implements(driverValuerType) {
		return appendDriverValuerValue
	}

	kind := typ.Kind()

	if kind != reflect.Ptr {
		ptr := reflect.PtrTo(typ)
		if ptr.Implements(queryAppenderType) {
			return addrAppender(appendQueryAppenderValue)
		}
		if ptr.Implements(driverValuerType) {
			return addrAppender(appendDriverValuerValue)
		}
	}

	switch kind {
	case reflect.Ptr:
		return ptrAppenderFunc(typ)
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return appendBytesValue
		}
	case reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return appendArrayBytesValue
		}
	}
	return appenders[kind]
}

func ptrAppenderFunc(typ reflect.Type) AppenderFunc {
	appender := Appender(typ.Elem())
	return func(fmter Formatter, b []byte, v reflect.Value) []byte {
		if v.IsNil() {
			return dialect.AppendNull(b)
		}
		return appender(fmter, b, v.Elem())
	}
}

func appendValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return dialect.AppendNull(b)
	}
	appender := Appender(v.Type())
	return appender(fmter, b, v)
}

func appendIfaceValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return Append(fmter, b, v.Interface())
}

func appendBoolValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return dialect.AppendBool(b, v.Bool())
}

func AppendIntValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return strconv.AppendInt(b, v.Int(), 10)
}

func AppendUintValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return strconv.AppendUint(b, v.Uint(), 10)
}

func appendFloat32Value(fmter Formatter, b []byte, v reflect.Value) []byte {
	return dialect.AppendFloat32(b, float32(v.Float()))
}

func appendFloat64Value(fmter Formatter, b []byte, v reflect.Value) []byte {
	return dialect.AppendFloat64(b, float64(v.Float()))
}

func appendBytesValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return dialect.AppendBytes(b, v.Bytes())
}

func appendArrayBytesValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	if v.CanAddr() {
		return dialect.AppendBytes(b, v.Slice(0, v.Len()).Bytes())
	}

	tmp := make([]byte, v.Len())
	reflect.Copy(reflect.ValueOf(tmp), v)
	b = dialect.AppendBytes(b, tmp)
	return b
}

func appendStringValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return dialect.AppendString(b, v.String())
}

func appendStructValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return appendJSONValue(fmter, b, v)
}

func appendJSONValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v.Interface()); err != nil {
		return dialect.AppendError(b, err)
	}

	bb := buf.Bytes()
	if len(bb) > 0 && bb[len(bb)-1] == '\n' {
		bb = bb[:len(bb)-1]
	}

	return dialect.AppendJSON(b, bb)
}

func appendTimeValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	tm := v.Interface().(time.Time)
	return dialect.AppendTime(b, tm)
}

func appendIPValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	ip := v.Interface().(net.IP)
	return dialect.AppendString(b, ip.String())
}

func appendIPNetValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	ipnet := v.Interface().(net.IPNet)
	return dialect.AppendString(b, ipnet.String())
}

func appendJSONRawMessageValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return dialect.AppendString(b, internal.String(v.Bytes()))
}

func appendQueryAppenderValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return AppendQueryAppender(fmter, b, v.Interface().(QueryAppender))
}

func appendDriverValuerValue(fmter Formatter, b []byte, v reflect.Value) []byte {
	return appendDriverValuer(fmter, b, v.Interface().(driver.Valuer))
}

func appendDriverValuer(fmter Formatter, b []byte, v driver.Valuer) []byte {
	value, err := v.Value()
	if err != nil {
		return dialect.AppendError(b, err)
	}
	return Append(fmter, b, value)
}

func addrAppender(fn AppenderFunc) AppenderFunc {
	return func(fmter Formatter, b []byte, v reflect.Value) []byte {
		if !v.CanAddr() {
			err := fmt.Errorf("bun: Append(nonaddressable %T)", v.Interface())
			return dialect.AppendError(b, err)
		}
		return fn(fmter, b, v.Addr())
	}
}
