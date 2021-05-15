package sqlfmt

import (
	"encoding/hex"
	"math"
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/uptrace/bun/internal/parser"
)

func Append(fmter Formatter, b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case nil:
		return AppendNull(b)
	case bool:
		return AppendBool(b, v)
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
		return AppendFloat32(b, v)
	case float64:
		return AppendFloat64(b, v)
	case string:
		return AppendString(b, v)
	case time.Time:
		return AppendTime(b, v)
	case []byte:
		return AppendBytes(b, v)
	case QueryAppender:
		return appendQueryAppender(fmter, b, v)
	default:
		return appendValue(fmter, b, reflect.ValueOf(v))
	}
}

func AppendError(b []byte, err error) []byte {
	b = append(b, "?!("...)
	b = append(b, err.Error()...)
	b = append(b, ')')
	return b
}

func AppendNull(b []byte) []byte {
	return append(b, "NULL"...)
}

func AppendBool(b []byte, v bool) []byte {
	if v {
		return append(b, "TRUE"...)
	}
	return append(b, "FALSE"...)
}

func AppendFloat32(b []byte, v float32) []byte {
	return appendFloat(b, float64(v), 32)
}

func AppendFloat64(b []byte, v float64) []byte {
	return appendFloat(b, v, 64)
}

func appendFloat(b []byte, v float64, bitSize int) []byte {
	switch {
	case math.IsNaN(v):
		return append(b, "'NaN'"...)
	case math.IsInf(v, 1):
		return append(b, "'Infinity'"...)
	case math.IsInf(v, -1):
		return append(b, "'-Infinity'"...)
	default:
		return strconv.AppendFloat(b, v, 'f', -1, bitSize)
	}
}

func AppendString(b []byte, s string) []byte {
	b = append(b, '\'')
	for _, c := range s {
		if c == '\000' {
			continue
		}

		if c == '\'' {
			b = append(b, '\'', '\'')
		} else {
			b = appendRune(b, c)
		}
	}
	b = append(b, '\'')
	return b
}

func appendRune(b []byte, r rune) []byte {
	if r < utf8.RuneSelf {
		return append(b, byte(r))
	}
	l := len(b)
	if cap(b)-l < utf8.UTFMax {
		b = append(b, make([]byte, utf8.UTFMax)...)
	}
	n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
	return b[:l+n]
}

func AppendBytes(b []byte, bytes []byte) []byte {
	if bytes == nil {
		return AppendNull(b)
	}

	b = append(b, `'\x`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bytes)))...)
	hex.Encode(b[s:], bytes)

	b = append(b, '\'')

	return b
}

func appendQueryAppender(fmter Formatter, b []byte, app QueryAppender) []byte {
	bb, err := app.AppendQuery(fmter, b)
	if err != nil {
		return AppendError(b, err)
	}
	return bb
}

func AppendTime(b []byte, tm time.Time) []byte {
	b = append(b, '\'')
	b = tm.UTC().AppendFormat(b, "2006-01-02 15:04:05.999999-07:00")
	b = append(b, '\'')
	return b
}

func AppendJSON(b, jsonb []byte) []byte {
	b = append(b, '\'')

	p := parser.New(jsonb)
	for p.Valid() {
		c := p.Read()
		switch c {
		case '"':
			b = append(b, '"')
		case '\'':
			b = append(b, "''"...)
		case '\000':
			continue
		case '\\':
			if p.SkipBytes([]byte("u0000")) {
				b = append(b, "\\\\u0000"...)
			} else {
				b = append(b, '\\')
				if p.Valid() {
					b = append(b, p.Read())
				}
			}
		default:
			b = append(b, c)
		}
	}

	b = append(b, '\'')

	return b
}
