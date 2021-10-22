package pgdialect

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type ArrayValue struct {
	v reflect.Value

	append schema.AppenderFunc
	scan   schema.ScannerFunc
}

// Array accepts a slice and returns a wrapper for working with PostgreSQL
// array data type.
//
// For struct fields you can use array tag:
//
//    Emails  []string `bun:",array"`
func Array(vi interface{}) *ArrayValue {
	v := reflect.ValueOf(vi)
	if !v.IsValid() {
		panic(fmt.Errorf("bun: Array(nil)"))
	}

	return &ArrayValue{
		v: v,

		append: pgDialect.arrayAppender(v.Type()),
		scan:   arrayScanner(v.Type()),
	}
}

var (
	_ schema.QueryAppender = (*ArrayValue)(nil)
	_ sql.Scanner          = (*ArrayValue)(nil)
)

func (a *ArrayValue) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	if a.append == nil {
		panic(fmt.Errorf("bun: Array(unsupported %s)", a.v.Type()))
	}
	return a.append(fmter, b, a.v), nil
}

func (a *ArrayValue) Scan(src interface{}) error {
	if a.scan == nil {
		return fmt.Errorf("bun: Array(unsupported %s)", a.v.Type())
	}
	if a.v.Kind() != reflect.Ptr {
		return fmt.Errorf("bun: Array(non-pointer %s)", a.v.Type())
	}
	return a.scan(a.v, src)
}

func (a *ArrayValue) Value() interface{} {
	if a.v.IsValid() {
		return a.v.Interface()
	}
	return nil
}
