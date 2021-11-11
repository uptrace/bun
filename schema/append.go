package schema

import (
	"reflect"
	"strconv"
	"time"

	"github.com/uptrace/bun/dialect"
)

func Append(fmter Formatter, b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case nil:
		return dialect.AppendNull(b)
	case bool:
		return dialect.AppendBool(b, v)
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, v, 10)
	case uint:
		return strconv.AppendInt(b, int64(v), 10)
	case uint32:
		return fmter.Dialect().AppendUint32(b, v)
	case uint64:
		return fmter.Dialect().AppendUint64(b, v)
	case float32:
		return dialect.AppendFloat32(b, v)
	case float64:
		return dialect.AppendFloat64(b, v)
	case string:
		return fmter.Dialect().AppendString(b, v)
	case time.Time:
		return fmter.Dialect().AppendTime(b, v)
	case []byte:
		return fmter.Dialect().AppendBytes(b, v)
	case QueryAppender:
		return AppendQueryAppender(fmter, b, v)
	default:
		vv := reflect.ValueOf(v)
		if vv.Kind() == reflect.Ptr && vv.IsNil() {
			return dialect.AppendNull(b)
		}
		appender := Appender(fmter.Dialect(), vv.Type())
		return appender(fmter, b, vv)
	}
}
