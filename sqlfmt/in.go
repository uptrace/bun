package sqlfmt

import (
	"fmt"
	"reflect"
)

type InValues struct {
	slice reflect.Value
	err   error
}

var _ QueryAppender = InValues{}

func In(slice interface{}) InValues {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return InValues{
			err: fmt.Errorf("bun: In(non-slice %T)", slice),
		}
	}
	return InValues{
		slice: v,
	}
}

func (in InValues) AppendQuery(fmter Formatter, b []byte) (_ []byte, err error) {
	if in.err != nil {
		return nil, in.err
	}
	return appendIn(fmter, b, in.slice), nil
}

func appendIn(fmter Formatter, b []byte, slice reflect.Value) []byte {
	sliceLen := slice.Len()
	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			b = append(b, ", "...)
		}

		elem := slice.Index(i)
		if elem.Kind() == reflect.Interface {
			elem = elem.Elem()
		}

		if elem.Kind() == reflect.Slice {
			b = append(b, '(')
			b = appendIn(fmter, b, elem)
			b = append(b, ')')
		} else {
			b = appendValue(fmter, b, elem)
		}
	}
	return b
}
