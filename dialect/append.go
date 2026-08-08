package dialect

import (
	"math"
	"strconv"

	"github.com/uptrace/bun/internal"
)

// StringEscaper appends s to b as a properly quoted and escaped SQL string
// literal. schema.Dialect satisfies this interface.
type StringEscaper interface {
	AppendString(b []byte, s string) []byte
}

// AppendError appends err inside a "?!(...)" marker so a formatting failure
// is visible in the generated query. err.Error() is untrusted: it can come
// from a driver.Valuer, JSON/msgpack encoder, etc., so it must go through
// escaper instead of being appended raw, or a crafted message could break
// out of the marker and inject SQL.
func AppendError(b []byte, escaper StringEscaper, err error) []byte {
	b = append(b, "?!("...)
	b = escaper.AppendString(b, err.Error())
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

func AppendFloat32(b []byte, num float32) []byte {
	return appendFloat(b, float64(num), 32)
}

func AppendFloat64(b []byte, num float64) []byte {
	return appendFloat(b, num, 64)
}

func appendFloat(b []byte, num float64, bitSize int) []byte {
	switch {
	case math.IsNaN(num):
		return append(b, "'NaN'"...)
	case math.IsInf(num, 1):
		return append(b, "'Infinity'"...)
	case math.IsInf(num, -1):
		return append(b, "'-Infinity'"...)
	default:
		return strconv.AppendFloat(b, num, 'f', -1, bitSize)
	}
}

//------------------------------------------------------------------------------

func AppendName(b []byte, ident string, quote byte) []byte {
	return appendName(b, internal.Bytes(ident), quote)
}

func appendName(b, ident []byte, quote byte) []byte {
	b = append(b, quote)
	for _, c := range ident {
		if c == quote {
			b = append(b, quote, quote)
		} else {
			b = append(b, c)
		}
	}
	b = append(b, quote)
	return b
}

func AppendIdent(b []byte, name string, quote byte) []byte {
	return appendIdent(b, internal.Bytes(name), quote)
}

func appendIdent(b, name []byte, quote byte) []byte {
	var quoted bool
loop:
	for _, c := range name {
		switch c {
		case '*':
			if !quoted {
				b = append(b, '*')
				continue loop
			}
		case '.':
			if quoted {
				b = append(b, quote)
				quoted = false
			}
			b = append(b, '.')
			continue loop
		}

		if !quoted {
			b = append(b, quote)
			quoted = true
		}
		if c == quote {
			b = append(b, quote, quote)
		} else {
			b = append(b, c)
		}
	}
	if quoted {
		b = append(b, quote)
	}
	return b
}
