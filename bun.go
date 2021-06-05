package bun

import (
	"context"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type (
	Safe  = schema.Safe
	Ident = schema.Ident
)

type NullTime = schema.NullTime

type BaseModel struct{}

type (
	BeforeScanHook   = schema.BeforeScanHook
	AfterScanHook    = schema.AfterScanHook
	AfterSelectHook  = schema.AfterSelectHook
	BeforeInsertHook = schema.BeforeInsertHook
	AfterInsertHook  = schema.AfterInsertHook
	BeforeUpdateHook = schema.BeforeUpdateHook
	AfterUpdateHook  = schema.AfterUpdateHook
	BeforeDeleteHook = schema.BeforeDeleteHook
	AfterDeleteHook  = schema.AfterDeleteHook
)

// type BeforeSelectQueryHook interface {
// 	BeforeSelectQuery(ctx context.Context, query *SelectQuery) error
// }

// type AfterSelectQueryHook interface {
// 	AfterSelectQuery(ctx context.Context, query *SelectQuery) error
// }

// type BeforeInsertQueryHook interface {
// 	BeforeInsertQuery(ctx context.Context, query *InsertQuery) error
// }

// type AfterInsertQueryHook interface {
// 	AfterInsertQuery(ctx context.Context, query *InsertQuery) error
// }

// type BeforeUpdateQueryHook interface {
// 	BeforeUpdateQuery(ctx context.Context, query *UpdateQuery) error
// }

// type AfterUpdateQueryHook interface {
// 	AfterUpdateQuery(ctx context.Context, query *UpdateQuery) error
// }

// type BeforeDeleteQueryHook interface {
// 	BeforeDeleteQuery(ctx context.Context, query *DeleteQuery) error
// }

// type AfterDeleteQueryHook interface {
// 	AfterDeleteQuery(ctx context.Context, query *DeleteQuery) error
// }

type BeforeCreateTableQueryHook interface {
	BeforeCreateTableQuery(ctx context.Context, query *CreateTableQuery) error
}

type AfterCreateTableQueryHook interface {
	AfterCreateTableQuery(ctx context.Context, query *CreateTableQuery) error
}

type BeforeDropTableQueryHook interface {
	BeforeDropTableQuery(ctx context.Context, query *DropTableQuery) error
}

type AfterDropTableQueryHook interface {
	AfterDropTableQuery(ctx context.Context, query *DropTableQuery) error
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
