package pgdialect

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

func arrayScanner(typ reflect.Type) schema.ScannerFunc {
	kind := typ.Kind()
	if kind == reflect.Ptr {
		typ = typ.Elem()
		kind = typ.Kind()
	}

	switch kind {
	case reflect.Slice, reflect.Array:
		// ok:
	default:
		return nil
	}

	elemType := typ.Elem()

	if kind == reflect.Slice {
		switch elemType {
		case stringType:
			return scanStringSliceValue
		case intType:
			return scanIntSliceValue
		case int64Type:
			return scanInt64SliceValue
		case float64Type:
			return scanFloat64SliceValue
		}
	}

	scanElem := schema.Scanner(elemType)
	return func(dest reflect.Value, src interface{}) error {
		dest = reflect.Indirect(dest)
		if !dest.CanSet() {
			return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
		}

		kind := dest.Kind()

		if src == nil {
			if kind != reflect.Slice || !dest.IsNil() {
				dest.Set(reflect.Zero(dest.Type()))
			}
			return nil
		}

		if kind == reflect.Slice {
			if dest.IsNil() {
				dest.Set(reflect.MakeSlice(dest.Type(), 0, 0))
			} else if dest.Len() > 0 {
				dest.Set(dest.Slice(0, 0))
			}
		}

		s, err := toString(src)
		if err != nil {
			return err
		}

		p := newArrayParser(s)
		nextValue := internal.MakeSliceNextElemFunc(dest)
		for {
			elem, err := p.NextElem()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			elemValue := nextValue()
			if err := scanElem(elemValue, elem); err != nil {
				return err
			}
		}

		return nil
	}
}

func scanStringSliceValue(dest reflect.Value, src interface{}) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := decodeStringSlice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func decodeStringSlice(src interface{}) ([]string, error) {
	if src == nil {
		return nil, nil
	}

	s, err := toString(src)
	if err != nil {
		return nil, err
	}

	slice := make([]string, 0)

	p := newArrayParser(s)
	for {
		elem, err := p.NextElem()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		slice = append(slice, string(elem))
	}

	return slice, nil
}

func scanIntSliceValue(dest reflect.Value, src interface{}) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := decodeIntSlice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func decodeIntSlice(src interface{}) ([]int, error) {
	if src == nil {
		return nil, nil
	}

	s, err := toString(src)
	if err != nil {
		return nil, err
	}

	slice := make([]int, 0)

	p := newArrayParser(s)
	for {
		elem, err := p.NextElem()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if elem == "" {
			slice = append(slice, 0)
			continue
		}

		n, err := strconv.Atoi(elem)
		if err != nil {
			return nil, err
		}

		slice = append(slice, n)
	}

	return slice, nil
}

func scanInt64SliceValue(dest reflect.Value, src interface{}) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := decodeInt64Slice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func decodeInt64Slice(src interface{}) ([]int64, error) {
	if src == nil {
		return nil, nil
	}

	s, err := toString(src)
	if err != nil {
		return nil, err
	}

	slice := make([]int64, 0)

	p := newArrayParser(s)
	for {
		elem, err := p.NextElem()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if elem == "" {
			slice = append(slice, 0)
			continue
		}

		n, err := strconv.ParseInt(elem, 10, 64)
		if err != nil {
			return nil, err
		}

		slice = append(slice, n)
	}

	return slice, nil
}

func scanFloat64SliceValue(dest reflect.Value, src interface{}) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := scanFloat64Slice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func scanFloat64Slice(src interface{}) ([]float64, error) {
	if src == -1 {
		return nil, nil
	}

	s, err := toString(src)
	if err != nil {
		return nil, err
	}

	slice := make([]float64, 0)

	p := newArrayParser(s)
	for {
		elem, err := p.NextElem()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if elem == "" {
			slice = append(slice, 0)
			continue
		}

		n, err := strconv.ParseFloat(elem, 64)
		if err != nil {
			return nil, err
		}

		slice = append(slice, n)
	}

	return slice, nil
}

func toString(src interface{}) (string, error) {
	switch src := src.(type) {
	case string:
		return src, nil
	case []byte:
		return string(src), nil
	default:
		return "", fmt.Errorf("bun: got %T, wanted []byte or string", src)
	}
}
