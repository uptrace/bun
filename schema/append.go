package schema

import (
	"reflect"
	"strconv"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/internal"
)

func FieldAppender(dialect Dialect, field *Field) AppenderFunc {
	if field.Tag.HasOption("msgpack") {
		return appendMsgpack
	}
	return dialect.Appender(field.StructField.Type)
}

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
		return strconv.AppendUint(b, uint64(v), 10)
	case uint32:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint64:
		return strconv.AppendUint(b, v, 10)
	case float32:
		return dialect.AppendFloat32(b, v)
	case float64:
		return dialect.AppendFloat64(b, v)
	case string:
		return dialect.AppendString(b, v)
	case time.Time:
		return dialect.AppendTime(b, v)
	case []byte:
		return dialect.AppendBytes(b, v)
	case QueryAppender:
		return AppendQueryAppender(fmter, b, v)
	default:
		return appendValue(fmter, b, reflect.ValueOf(v))
	}
}

func appendMsgpack(fmter Formatter, b []byte, v reflect.Value) []byte {
	hexEnc := internal.NewHexEncoder(b)

	enc := msgpack.GetEncoder()
	defer msgpack.PutEncoder(enc)

	enc.Reset(hexEnc)
	if err := enc.EncodeValue(v); err != nil {
		return dialect.AppendError(b, err)
	}

	if err := hexEnc.Close(); err != nil {
		return dialect.AppendError(b, err)
	}

	return hexEnc.Bytes()
}

func AppendQueryAppender(fmter Formatter, b []byte, app QueryAppender) []byte {
	bb, err := app.AppendQuery(fmter, b)
	if err != nil {
		return dialect.AppendError(b, err)
	}
	return bb
}
