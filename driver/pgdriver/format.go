package pgdriver

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

func formatQueryArgs(query string, args []interface{}) (string, error) {
	namedArgs := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		namedArgs[i] = driver.NamedValue{Value: arg}
	}
	return formatQuery(query, namedArgs)
}

func formatQuery(query string, args []driver.NamedValue) (string, error) {
	if len(args) == 0 {
		return query, nil
	}

	dst := make([]byte, 0, 2*len(query))

	p := newParser(query)
	for p.Valid() {
		switch c := p.Next(); c {
		case '$':
			if i, ok := p.Number(); ok {
				if i < 1 {
					return "", fmt.Errorf("pgdriver: got $%d, but the minimal arg index is 1", i)
				}
				if i > len(args) {
					return "", fmt.Errorf("pgdriver: got %d args, wanted %d", len(args), i)
				}

				var err error
				dst, err = appendArg(dst, args[i-1].Value)
				if err != nil {
					return "", err
				}
			} else {
				dst = append(dst, '$')
			}
		case '\'':
			if b, ok := p.QuotedString(); ok {
				dst = append(dst, b...)
			} else {
				dst = append(dst, '\'')
			}
		default:
			dst = append(dst, c)
		}
	}

	return bytesToString(dst), nil
}

func appendArg(b []byte, v interface{}) ([]byte, error) {
	switch v := v.(type) {
	case nil:
		return append(b, "NULL"...), nil
	case int64:
		return strconv.AppendInt(b, v, 10), nil
	case float64:
		switch {
		case math.IsNaN(v):
			return append(b, "'NaN'"...), nil
		case math.IsInf(v, 1):
			return append(b, "'Infinity'"...), nil
		case math.IsInf(v, -1):
			return append(b, "'-Infinity'"...), nil
		default:
			return strconv.AppendFloat(b, v, 'f', -1, 64), nil
		}
	case bool:
		if v {
			return append(b, "TRUE"...), nil
		}
		return append(b, "FALSE"...), nil
	case []byte:
		if v == nil {
			return append(b, "NULL"...), nil
		}

		b = append(b, `'\x`...)

		s := len(b)
		b = append(b, make([]byte, hex.EncodedLen(len(v)))...)
		hex.Encode(b[s:], v)

		b = append(b, "'"...)

		return b, nil
	case string:
		b = append(b, '\'')
		for _, r := range v {
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
		return b, nil
	case time.Time:
		if v.IsZero() {
			return append(b, "NULL"...), nil
		}
		return v.UTC().AppendFormat(b, "'2006-01-02 15:04:05.999999-07:00'"), nil
	default:
		return nil, fmt.Errorf("pgdriver: unexpected arg: %T", v)
	}
}

type parser struct {
	b []byte
	i int
}

func newParser(s string) *parser {
	return &parser{
		b: stringToBytes(s),
	}
}

func (p *parser) Valid() bool {
	return p.i < len(p.b)
}

func (p *parser) Next() byte {
	c := p.b[p.i]
	p.i++
	return c
}

func (p *parser) Number() (int, bool) {
	start := p.i
	end := len(p.b)

	for i := p.i; i < len(p.b); i++ {
		c := p.b[i]
		if !isNum(c) {
			end = i
			break
		}
	}

	p.i = end
	b := p.b[start:end]

	n, err := strconv.Atoi(bytesToString(b))
	if err != nil {
		return 0, false
	}

	return n, true
}

func (p *parser) QuotedString() ([]byte, bool) {
	start := p.i - 1
	end := len(p.b)

	var c byte
	for i := p.i; i < len(p.b); i++ {
		next := p.b[i]
		if c == '\'' && next != '\'' {
			end = i
			break
		}
		c = next
	}

	p.i = end
	b := p.b[start:end]

	return b, true
}

func isNum(c byte) bool {
	return c >= '0' && c <= '9'
}
