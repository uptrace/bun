package pgdialect

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/uptrace/bun/internal"
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
	src, ok := anySrc.([]byte)
	if !ok {
		return fmt.Errorf("pgdialect: Range can't scan %T", anySrc)
	}

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
