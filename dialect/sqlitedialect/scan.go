package sqlitedialect

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

func scanner(typ reflect.Type) schema.ScannerFunc {
	if typ.Kind() == reflect.Interface {
		return scanInterface
	}
	return schema.Scanner(typ)
}

func scanInterface(dest reflect.Value, src interface{}) error {
	if dest.IsNil() {
		dest.Set(reflect.ValueOf(src))
		return nil
	}

	dest = dest.Elem()
	if fn := scanner(dest.Type()); fn != nil {
		return fn(dest, src)
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}
