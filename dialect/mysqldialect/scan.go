package mysqldialect

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

func scanner(typ reflect.Type) schema.ScannerFunc {
	switch typ.Kind() {
	case reflect.Interface:
		return scanInterface
	case reflect.Bool:
		return scanBool
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

func scanBool(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case []byte:
		dest.SetBool(src[0] != 48)
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}
