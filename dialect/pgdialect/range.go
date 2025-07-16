package pgdialect

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type Range[T any] struct {
	Lower, Upper           T
	LowerBound, UpperBound RangeBound
}

type MultiRange[T any] []Range[T]

type RangeBound byte

const (
	RangeBoundInclusiveLeft  RangeBound = '['
	RangeBoundInclusiveRight RangeBound = ']'
	RangeBoundExclusiveLeft  RangeBound = '('
	RangeBoundExclusiveRight RangeBound = ')'
	RangeBoundUnbound        RangeBound = 'U' // U is just a placeholder and has no actual meaning
	RangeBoundEmpty          RangeBound = 'E' // E is just a placeholder and has no actual meaning
)

type RangeOption[T any] func(*Range[T])

func NewRange[T any](lower, upper T, opts ...RangeOption[T]) Range[T] {
	r := Range[T]{
		Lower:      lower,
		Upper:      upper,
		LowerBound: RangeBoundInclusiveLeft,
		UpperBound: RangeBoundExclusiveRight,
	}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

func WithLowerBound[T any](bound RangeBound) RangeOption[T] {
	return func(r *Range[T]) {
		r.LowerBound = bound
	}
}

func WithUpperBound[T any](bound RangeBound) RangeOption[T] {
	return func(r *Range[T]) {
		r.UpperBound = bound
	}
}

func NewRangeEmpty[T any]() Range[T] {
	return Range[T]{LowerBound: RangeBoundEmpty, UpperBound: RangeBoundEmpty}
}

func (r *Range[T]) IsZero() bool {
	return r == nil || r.LowerBound == 0
}

func (r Range[T]) IsEmpty() bool {
	return r.LowerBound == RangeBoundEmpty
}

var _ sql.Scanner = (*Range[any])(nil)

func (r *Range[T]) Scan(raw any) (err error) {
	var src []byte
	switch v := raw.(type) {
	case []byte:
		src = v
	case string:
		src = []byte(v)
	case nil:
		return nil
	default:
		return fmt.Errorf("pgdialect: Range can't scan %T", raw)
	}

	src = bytes.TrimSpace(src)
	if len(src) == 0 {
		return nil
	}

	if string(src) == "empty" {
		r.LowerBound, r.UpperBound = RangeBoundEmpty, RangeBoundEmpty
		return nil
	}

	switch src[0] {
	case byte(RangeBoundInclusiveLeft), byte(RangeBoundExclusiveLeft):
		r.LowerBound = RangeBound(src[0])
	default:
		return fmt.Errorf("unexpected lower bound: %s", string(src[:1]))
	}
	switch src[len(src)-1] {
	case byte(RangeBoundInclusiveRight), byte(RangeBoundExclusiveRight):
		r.UpperBound = RangeBound(src[len(src)-1])
	default:
		return fmt.Errorf("unexpected upper bound: %s", string(src[len(src)-1:]))
	}

	src = src[1 : len(src)-1]

	ind := bytes.IndexByte(src, ',')
	if ind == -1 {
		return fmt.Errorf("invalid range: wanted comma, got %s", string(src))
	}
	left, right := src[:ind], src[ind+1:]

	if len(left) > 0 {
		_, err := scanElem(&r.Lower, left)
		if err != nil {
			return err
		}
	} else {
		r.LowerBound = RangeBoundUnbound
	}

	if len(right) > 0 {
		_, err = scanElem(&r.Upper, right)
		if err != nil {
			return err
		}
	} else {
		r.UpperBound = RangeBoundUnbound
	}

	return nil
}

var _ schema.QueryAppender = (*Range[any])(nil)

func (r Range[T]) AppendQuery(_ schema.Formatter, buf []byte) ([]byte, error) {
	buf = append(buf, '\'')
	buf = appendRange(buf, r)
	buf = append(buf, '\'')
	return buf, nil
}

func appendRange[T any](buf []byte, r Range[T]) []byte {
	if r.IsEmpty() {
		buf = append(buf, []byte("empty")...)
		return buf
	}

	if r.LowerBound == RangeBoundUnbound {
		buf = append(buf, byte(RangeBoundExclusiveLeft))
	} else {
		buf = append(buf, byte(r.LowerBound))
		buf = appendElem(buf, r.Lower)
	}
	buf = append(buf, ',')
	if r.UpperBound == RangeBoundUnbound {
		buf = append(buf, byte(RangeBoundExclusiveRight))
	} else {
		buf = appendElem(buf, r.Upper)
		buf = append(buf, byte(r.UpperBound))
	}
	return buf
}

func (m *MultiRange[T]) IsZero() bool {
	return m == nil || len(([]Range[T])(*m)) == 0
}

func (m MultiRange[T]) AppendQuery(_ schema.Formatter, buf []byte) ([]byte, error) {
	if m == nil {
		return append(buf, []byte("'{}'")...), nil
	}
	rs := ([]Range[T])(m)
	buf = append(buf, '\'', '{')
	for _, r := range rs {
		buf = appendRange(buf, r)
		buf = append(buf, ',')
	}
	if len(rs) > 0 {
		buf[len(buf)-1] = '}'
	} else {
		buf = append(buf, '}')
	}
	buf = append(buf, '\'')
	return buf, nil
}

func scanElem(ptr any, src []byte) ([]byte, error) {
	// NOTE: for daterange, pg return 2024-12-01, for tzrange, pg return "2024-12-01 12:00:00"
	if len(src) >= 2 && src[0] == '"' {
		src = src[1 : len(src)-1]
	}

	switch ptr := ptr.(type) {
	case *time.Time:
		tm, err := internal.ParseTime(internal.String(src))
		if err != nil {
			return nil, err
		}
		*ptr = tm

		return src, nil

	case sql.Scanner:
		if err := ptr.Scan(src); err != nil {
			return nil, err
		}
		return src, nil

	default:
		panic(fmt.Errorf("unsupported range type: %T", ptr))
	}
}
