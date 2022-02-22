package bun

import (
	"context"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type (
	Safe  = schema.Safe
	Ident = schema.Ident

	NullTime  = schema.NullTime
	BaseModel = schema.BaseModel
	Query     = schema.Query

	BeforeAppendModelHook = schema.BeforeAppendModelHook

	BeforeScanRowHook = schema.BeforeScanRowHook
	AfterScanRowHook  = schema.AfterScanRowHook

	// DEPRECATED. Use BeforeScanRowHook instead.
	BeforeScanHook = schema.BeforeScanHook
	// DEPRECATED. Use AfterScanRowHook instead.
	AfterScanHook = schema.AfterScanHook
)

type BeforeSelectHook interface {
	BeforeSelect(ctx context.Context, query *SelectQuery) error
}

type AfterSelectHook interface {
	AfterSelect(ctx context.Context, query *SelectQuery) error
}

type BeforeInsertHook interface {
	BeforeInsert(ctx context.Context, query *InsertQuery) error
}

type AfterInsertHook interface {
	AfterInsert(ctx context.Context, query *InsertQuery) error
}

type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context, query *UpdateQuery) error
}

type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context, query *UpdateQuery) error
}

type BeforeDeleteHook interface {
	BeforeDelete(ctx context.Context, query *DeleteQuery) error
}

type AfterDeleteHook interface {
	AfterDelete(ctx context.Context, query *DeleteQuery) error
}

type BeforeCreateTableHook interface {
	BeforeCreateTable(ctx context.Context, query *CreateTableQuery) error
}

type AfterCreateTableHook interface {
	AfterCreateTable(ctx context.Context, query *CreateTableQuery) error
}

type BeforeDropTableHook interface {
	BeforeDropTable(ctx context.Context, query *DropTableQuery) error
}

type AfterDropTableHook interface {
	AfterDropTable(ctx context.Context, query *DropTableQuery) error
}

// SetLogger overwriters default Bun logger.
func SetLogger(logger internal.Logging) {
	internal.Logger = logger
}

// SetWarnLogger overwriters default Bun warn logger.
func SetWarnLogger(logger internal.Logging) {
	internal.Warn = logger
}

//------------------------------------------------------------------------------

type InValues struct {
	slice reflect.Value
	err   error
}

var _ schema.QueryAppender = InValues{}

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

func (in InValues) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if in.err != nil {
		return nil, in.err
	}
	return appendIn(fmter, b, in.slice), nil
}

func appendIn(fmter schema.Formatter, b []byte, slice reflect.Value) []byte {
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
			b = fmter.AppendValue(b, elem)
		}
	}
	return b
}
