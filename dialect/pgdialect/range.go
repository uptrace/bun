package pgdialect

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type MultiRange[T any] []Range[T]

type Range[T any] struct {
	Lower, Upper           *T
	LowerBound, UpperBound RangeBound
	isEmpty                bool
}

func (r Range[T]) IsEmpty() bool {
	return r.isEmpty
}

type RangeBound byte

const (
	RangeBoundInclusiveLeft  RangeBound = '['
	RangeBoundInclusiveRight RangeBound = ']'
	RangeBoundExclusiveLeft  RangeBound = '('
	RangeBoundExclusiveRight RangeBound = ')'
)

func NewRange[T any](lower, upper *T) Range[T] {
	return Range[T]{
		Lower:      lower,
		Upper:      upper,
		LowerBound: RangeBoundInclusiveLeft,
		UpperBound: RangeBoundExclusiveRight,
	}
}

func NewEmptyRange[T any]() Range[T] {
	return Range[T]{isEmpty: true}
}

var _ sql.Scanner = (*Range[any])(nil)

func (r *Range[T]) Scan(raw any) (err error) {
	src, ok := raw.([]byte)
	if !ok {
		return fmt.Errorf("pgdialect: Range can't scan %T", raw)
	}

	if len(src) == 0 {
		return io.ErrUnexpectedEOF
	}
	if string(src) == "empty" {
		r.isEmpty = true
		return nil
	}

	r.LowerBound = RangeBound(src[0])

	ind := bytes.IndexByte(src, ',')
	if ind == -1 {
		return fmt.Errorf("invalid range: wanted comma, got %s", string(src))
	}
	left, right := src[1:ind], src[ind+1:len(src)-1]

	if len(left) > 0 {
		_, err := scanElem(r.Lower, left)
		if err != nil {
			return err
		}
	}

	if len(right) > 0 {
		_, err = scanElem(r.Upper, right)
		if err != nil {
			return err
		}
	}

	r.UpperBound = RangeBound(src[len(src)-1])
	return nil
}

var _ schema.QueryAppender = (*Range[any])(nil)

func (r *Range[T]) AppendQuery(fmt schema.Formatter, buf []byte) ([]byte, error) {
	if r.isEmpty {
		buf = append(buf, []byte("'empty'")...)
		return buf, nil
	}

	buf = append(buf, '\'', byte(r.LowerBound))
	if r.Lower != nil {
		buf = appendElem(buf, *r.Lower)
	}
	buf = append(buf, ',')
	if r.Upper != nil {
		buf = appendElem(buf, *r.Upper)
	}
	buf = append(buf, byte(r.UpperBound), '\'')
	return buf, nil
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

	case sql.Scanner:
		src, str, err := readStringLiteral(src)
		if err != nil {
			return nil, err
		}
		if err := ptr.Scan(str); err != nil {
			return nil, err
		}
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
