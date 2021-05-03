package schema

import (
	"reflect"

	"github.com/uptrace/bun/sqlfmt"
	"github.com/vmihailenco/msgpack/v5"
)

func FieldAppender(field *Field) sqlfmt.AppenderFunc {
	if field.Tag.HasOption("msgpack") {
		return appendMsgpack
	}
	return sqlfmt.Appender(field.Type)
}

func appendMsgpack(fmter sqlfmt.QueryFormatter, b []byte, v reflect.Value) []byte {
	hexEnc := sqlfmt.NewHexEncoder(b)

	enc := msgpack.GetEncoder()
	defer msgpack.PutEncoder(enc)

	enc.Reset(hexEnc)
	if err := enc.EncodeValue(v); err != nil {
		return sqlfmt.AppendError(b, err)
	}

	if err := hexEnc.Close(); err != nil {
		return sqlfmt.AppendError(b, err)
	}

	return hexEnc.Bytes()
}
