package dialect

import (
	"encoding/hex"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/parser"
)

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
	for _, r := range s {
		if r == '\000' {
			continue
		}

		if r == '\'' {
			b = append(b, '\'', '\'')
			continue
		}

		if r < utf8.RuneSelf {
			b = append(b, byte(r))
			continue
		}

		l := len(b)
		if cap(b)-l < utf8.UTFMax {
			b = append(b, make([]byte, utf8.UTFMax)...)
		}
		n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
		b = b[:l+n]
	}
	b = append(b, '\'')
	return b
}

func AppendBytes(b []byte, bs []byte) []byte {
	if bs == nil {
		return AppendNull(b)
	}

	b = append(b, `'\x`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

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
				b = append(b, `\\u0000`...)
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

//------------------------------------------------------------------------------

func AppendIdent(b []byte, field string, quote byte) []byte {
	return appendIdent(b, internal.Bytes(field), quote)
}

func appendIdent(b, src []byte, quote byte) []byte {
	var quoted bool
loop:
	for _, c := range src {
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
