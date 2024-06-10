package pgdialect

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/parser"
	"github.com/uptrace/bun/schema"
)

type MultiRange[T any] []Range[T]

type Range[T any] struct {
	Lower, Upper           T
	LowerBound, UpperBound RangeBound
}

type RangeBound byte

const (
	RangeBoundInclusiveLeft  RangeBound = '['
	RangeBoundInclusiveRight RangeBound = ']'
	RangeBoundExclusiveLeft  RangeBound = '('
	RangeBoundExclusiveRight RangeBound = ')'
)

func NewRange[T any](lower, upper T) Range[T] {
	return Range[T]{
		Lower:      lower,
		Upper:      upper,
		LowerBound: RangeBoundInclusiveLeft,
		UpperBound: RangeBoundExclusiveRight,
	}
}

var _ sql.Scanner = (*Range[any])(nil)

func (r *Range[T]) Scan(anySrc any) (err error) {
	src := anySrc.([]byte)

	if len(src) == 0 {
		return io.ErrUnexpectedEOF
	}
	r.LowerBound = RangeBound(src[0])
	src = src[1:]

	src, err = scanElem(&r.Lower, src)
	if err != nil {
		return err
	}

	if len(src) == 0 {
		return io.ErrUnexpectedEOF
	}
	if ch := src[0]; ch != ',' {
		return fmt.Errorf("got %q, wanted %q", ch, ',')
	}
	src = src[1:]

	src, err = scanElem(&r.Upper, src)
	if err != nil {
		return err
	}

	if len(src) == 0 {
		return io.ErrUnexpectedEOF
	}
	r.UpperBound = RangeBound(src[0])
	src = src[1:]

	if len(src) > 0 {
		return fmt.Errorf("unread data: %q", src)
	}
	return nil
}

var _ schema.QueryAppender = (*Range[any])(nil)

func (r *Range[T]) AppendQuery(fmt schema.Formatter, buf []byte) ([]byte, error) {
	buf = append(buf, byte(r.LowerBound))
	buf = appendElem(buf, r.Lower)
	buf = append(buf, ',')
	buf = appendElem(buf, r.Upper)
	buf = append(buf, byte(r.UpperBound))
	return buf, nil
}

func appendElem(buf []byte, val any) []byte {
	switch val := val.(type) {
	case time.Time:
		buf = append(buf, '"')
		buf = appendTime(buf, val)
		buf = append(buf, '"')
		return buf
	default:
		panic(fmt.Errorf("unsupported range type: %T", val))
	}
}

func scanElem(ptr any, src []byte) ([]byte, error) {
	switch ptr := ptr.(type) {
	case *time.Time:
		src, str, err := readStringLiteral(src)
		if err != nil {
			return nil, err
		}

		tm, err := internal.ParseTime(internal.String(str))
		if err != nil {
			return nil, err
		}
		*ptr = tm

		return src, nil
	default:
		panic(fmt.Errorf("unsupported range type: %T", ptr))
	}
}

func readStringLiteral(src []byte) ([]byte, []byte, error) {
	p := newParser(src)

	if err := p.Skip('"'); err != nil {
		return nil, nil, err
	}

	str, err := p.ReadSubstring('"')
	if err != nil {
		return nil, nil, err
	}

	src = p.Remaining()
	return src, str, nil
}

//------------------------------------------------------------------------------

type pgparser struct {
	parser.Parser
	buf []byte
}

func newParser(b []byte) *pgparser {
	p := new(pgparser)
	p.Reset(b)
	return p
}

func (p *pgparser) ReadLiteral(ch byte) []byte {
	p.Unread()
	lit, _ := p.ReadSep(',')
	return lit
}

func (p *pgparser) ReadUnescapedSubstring(ch byte) ([]byte, error) {
	return p.readSubstring(ch, false)
}

func (p *pgparser) ReadSubstring(ch byte) ([]byte, error) {
	return p.readSubstring(ch, true)
}

func (p *pgparser) readSubstring(ch byte, escaped bool) ([]byte, error) {
	ch, err := p.ReadByte()
	if err != nil {
		return nil, err
	}

	p.buf = p.buf[:0]
	for {
		if ch == '"' {
			break
		}

		next, err := p.ReadByte()
		if err != nil {
			return nil, err
		}

		if ch == '\\' {
			switch next {
			case '\\', '"':
				p.buf = append(p.buf, next)

				ch, err = p.ReadByte()
				if err != nil {
					return nil, err
				}
			default:
				p.buf = append(p.buf, '\\')
				ch = next
			}
			continue
		}

		if escaped && ch == '\'' && next == '\'' {
			p.buf = append(p.buf, next)
			ch, err = p.ReadByte()
			if err != nil {
				return nil, err
			}
			continue
		}

		p.buf = append(p.buf, ch)
		ch = next
	}

	if bytes.HasPrefix(p.buf, []byte("\\x")) && len(p.buf)%2 == 0 {
		data := p.buf[2:]
		buf := make([]byte, hex.DecodedLen(len(data)))
		n, err := hex.Decode(buf, data)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}

	return p.buf, nil
}

func (p *pgparser) ReadRange(ch byte) ([]byte, error) {
	p.buf = p.buf[:0]
	p.buf = append(p.buf, ch)

	for p.Valid() {
		ch = p.Read()
		p.buf = append(p.buf, ch)
		if ch == ']' || ch == ')' {
			break
		}
	}

	return p.buf, nil
}
