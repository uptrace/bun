package pgdialect

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type MultiRange[T any] []Range[T]

var (
	_ sql.Scanner   = (*MultiRange[any])(nil)
	_ driver.Valuer = (*MultiRange[any])(nil)
)

func (m *MultiRange[T]) Scan(anySrc any) (err error) {
	return Array(m).Scan(anySrc)
}

func (m MultiRange[T]) Value() (driver.Value, error) {
	return m.String(), nil
}

func (m MultiRange[T]) String() string {
	if len(m) == 0 {
		return "{}"
	}
	var b []byte
	b = append(b, '{')
	for _, r := range m {
		b = append(b, unquote(appendElem(nil, r))...)
		b = append(b, ',')
	}
	b = append(b[:len(b)-1], '}')
	return string(b)
}

type Range[T any] struct {
	Lower, Upper T
	LowerBound   RangeLowerBound
	UpperBound   RangeUpperBound
}

var (
	_ driver.Valuer = (*Range[any])(nil)
	_ sql.Scanner   = (*Range[any])(nil)
)

type RangeLowerBound byte
type RangeUpperBound byte

const (
	RangeBoundExclusiveLeft  RangeLowerBound = '('
	RangeBoundExclusiveRight RangeUpperBound = ')'
	RangeBoundInclusiveLeft  RangeLowerBound = '['
	RangeBoundInclusiveRight RangeUpperBound = ']'

	RangeBoundDefaultLeft  = RangeBoundInclusiveLeft
	RangeBoundDefaultRight = RangeBoundExclusiveRight
)

func NewRange[T any](lower, upper T) Range[T] {
	return Range[T]{
		Lower:      lower,
		LowerBound: RangeBoundDefaultLeft,
		Upper:      upper,
		UpperBound: RangeBoundDefaultRight,
	}
}

func (r *Range[T]) Scan(src any) (err error) {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("unsupported data type: %T", src)
	}

	if len(srcB) == 0 {
		return fmt.Errorf("invalid format: %s", string(srcB))
	}

	if string(srcB) == "empty" {
		return nil
	}

	// read bounds
	r.LowerBound = RangeLowerBound(srcB[0])
	r.UpperBound = RangeUpperBound(srcB[len(srcB)-1])
	srcB = srcB[1 : len(srcB)-1]
	if len(srcB) == 0 {
		return fmt.Errorf("invalid format: %s", string(srcB))
	}

	l, u, ok := bytes.Cut(srcB, []byte(","))
	if !ok {
		return fmt.Errorf("invalid format: %s", string(srcB))
	}

	scanner := schema.Scanner(reflect.TypeOf(r.Lower))
	if err := scanner(reflect.ValueOf(&r.Lower).Elem(), unquote(l)); err != nil {
		return err
	}
	if err := scanner(reflect.ValueOf(&r.Upper).Elem(), unquote(u)); err != nil {
		return err
	}
	return nil
}

func (r Range[T]) Value() (driver.Value, error) {
	return r.String(), nil
}

func (r Range[T]) String() string {
	if r.IsZero() {
		return "empty"
	}
	var rs []byte
	if r.LowerBound == 0 {
		rs = append(rs, byte(RangeBoundDefaultLeft))
	} else {
		rs = append(rs, byte(r.LowerBound))
	}
	rs = appendElem(rs, r.Lower)
	rs = append(rs, ',')
	rs = appendElem(rs, r.Upper)
	if r.UpperBound == 0 {
		rs = append(rs, byte(RangeBoundDefaultRight))
	} else {
		rs = append(rs, byte(r.UpperBound))
	}
	return string(rs)
}

func (r Range[T]) IsZero() bool {
	return r.LowerBound == 0 && r.UpperBound == 0
}

func unquote(s []byte) []byte {
	if len(s) == 0 {
		return s
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		return bytes.ReplaceAll(s[1:len(s)-1], []byte("\\\""), []byte("\""))
	}
	return s
}
