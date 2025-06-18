package pgdialect

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type MultiRange[T any] []Range[T]

var (
	_ sql.Scanner          = (*MultiRange[any])(nil)
	_ schema.QueryAppender = (*MultiRange[any])(nil)
)

func (m *MultiRange[T]) Scan(anySrc any) (err error) {
	return Array(m).Scan(anySrc)
}

func (m MultiRange[T]) AppendQuery(fmt schema.Formatter, buf []byte) ([]byte, error) {
	ms := "{"
	for _, r := range m {
		ms += r.String() + ","
	}
	ms = ms[:len(ms)-1] + "}"
	// to put between simple quote
	return fmt.AppendQuery(buf, "?", ms), nil
}

type Range[T any] struct {
	Lower, Upper T
	LowerBound   RangeLowerBound
	UpperBound   RangeUpperBound
}

type RangeLowerBound byte
type RangeUpperBound byte

const (
	RangeLowerBoundInclusive RangeLowerBound = '['
	RangeLowerBoundExclusive RangeLowerBound = '('
	RangeUpperBoundInclusive RangeUpperBound = ']'
	RangeUpperBoundExclusive RangeUpperBound = ')'

	RangeLowerBoundDefault = RangeLowerBoundInclusive
	RangeUpperBoundDefault = RangeUpperBoundExclusive
)

func NewRange[T any](lower, upper T) Range[T] {
	return Range[T]{
		Lower:      lower,
		Upper:      upper,
		LowerBound: RangeLowerBoundDefault,
		UpperBound: RangeUpperBoundDefault,
	}
}

var (
	_ sql.Scanner          = (*Range[any])(nil)
	_ schema.QueryAppender = (*Range[any])(nil)
)

func (r Range[T]) IsZero() bool {
	return r.LowerBound == 0 && r.UpperBound == 0 && reflect.ValueOf(r.Lower).IsZero() && reflect.ValueOf(r.Upper).IsZero()
}

func (r *Range[T]) Scan(anySrc any) (err error) {
	var src []byte
	switch s := anySrc.(type) {
	case string:
		src = []byte(s)
	case []byte:
		src = s
	default:
		return fmt.Errorf("pgdialect: Range can't scan %T", anySrc)
	}

	src = bytes.TrimSpace(src)
	if len(src) == 0 {
		return io.ErrUnexpectedEOF
	}

	if string(src) == "empty" {
		return nil
	}

	// read bounds
	r.LowerBound = RangeLowerBound(src[0])
	r.UpperBound = RangeUpperBound(src[len(src)-1])
	src = src[1 : len(src)-1]
	if len(src) == 0 {
		return io.ErrUnexpectedEOF
	}

	// read range
	src, err = scanElem(&r.Lower, src)
	if err != nil {
		return err
	}
	src, err = scanElem(&r.Upper, src)
	if err != nil {
		return err
	}

	if len(src) > 0 {
		return fmt.Errorf("unread data: %q", src)
	}
	return nil
}

var _ schema.QueryAppender = (*Range[any])(nil)

func (r Range[T]) String() string {
	var rs []byte
	if r.LowerBound == 0 {
		rs = append(rs, byte(RangeLowerBoundDefault))
	} else {
		rs = append(rs, byte(r.LowerBound))
	}
	rs = appendElem(rs, r.Lower)
	rs = append(rs, ',')
	rs = appendElem(rs, r.Upper)
	if r.UpperBound == 0 {
		rs = append(rs, byte(RangeUpperBoundDefault))
	} else {
		rs = append(rs, byte(r.UpperBound))
	}
	return string(rs)
}

func (r Range[T]) AppendQuery(fmt schema.Formatter, buf []byte) ([]byte, error) {
	// to put between simple quote
	return fmt.AppendQuery(buf, "?", r.String()), nil
}

// scanElem scan range
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

// readStringLiteral split range sperator
func readStringLiteral(src []byte) ([]byte, []byte, error) {
	i := bytes.IndexRune(src, ',')
	if i == -1 {
		return nil, bytes.Trim(src, "\""), nil
	}
	return src[i+1:], bytes.Trim(src[:i], "\""), nil
}

type NullRange[T any] struct {
	Range Range[T]
	Valid bool
}

func (n *NullRange[T]) Scan(value any) error {
	if value == nil {
		n.Range, n.Valid = Range[T]{}, false
		return nil
	}
	n.Valid = true
	return n.Range.Scan(value)
}

func (n NullRange[T]) AppendQuery(fmt schema.Formatter, buf []byte) ([]byte, error) {
	if !n.Valid {
		return dialect.AppendNull(buf), nil
	}
	return n.Range.AppendQuery(fmt, buf)
}
