package pgdialect

import (
	"fmt"
	"io"
	"reflect"

	"github.com/uptrace/bun/schema"
)

func hstoreScanner(typ reflect.Type) schema.ScannerFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := hstoreScanner(typ.Elem()); fn != nil {
			return schema.PtrScanner(fn)
		}
	case reflect.Map:
		// ok:
	default:
		return nil
	}

	if typ.Key() == stringType && typ.Elem() == stringType {
		return scanMapStringStringValue
	}
	return func(dest reflect.Value, src interface{}) error {
		return fmt.Errorf("bun: Hstore(unsupported %s)", dest.Type())
	}
}

func scanMapStringStringValue(dest reflect.Value, src interface{}) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	m, err := decodeMapStringString(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(m))
	return nil
}

func decodeMapStringString(src interface{}) (map[string]string, error) {
	if src == nil {
		return nil, nil
	}

	b, err := toBytes(src)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)

	p := newHStoreParser(b)
	for {
		key, err := p.NextKey()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		value, err := p.NextValue()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		m[key] = value
	}

	return m, nil
}
