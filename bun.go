package bun

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type (
	Safe  = schema.Safe
	Ident = schema.Ident
)

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

var jsonNull = []byte("null")

// NullTime is a time.Time wrapper that marshals zero time as JSON null and SQL NULL.
type NullTime struct {
	time.Time
}

var (
	_ json.Marshaler       = (*NullTime)(nil)
	_ json.Unmarshaler     = (*NullTime)(nil)
	_ sql.Scanner          = (*NullTime)(nil)
	_ schema.QueryAppender = (*NullTime)(nil)
)

func (tm NullTime) MarshalJSON() ([]byte, error) {
	if tm.IsZero() {
		return jsonNull, nil
	}
	return tm.Time.MarshalJSON()
}

func (tm *NullTime) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, jsonNull) {
		tm.Time = time.Time{}
		return nil
	}
	return tm.Time.UnmarshalJSON(b)
}

func (tm NullTime) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	if tm.IsZero() {
		return dialect.AppendNull(b), nil
	}
	return dialect.AppendTime(b, tm.Time), nil
}

func (tm *NullTime) Scan(src interface{}) error {
	if src == nil {
		tm.Time = time.Time{}
		return nil
	}

	switch src := src.(type) {
	case []byte:
		newtm, err := internal.ParseTime(internal.String(src))
		if err != nil {
			return err
		}

		tm.Time = newtm
		return nil
	case time.Time:
		tm.Time = src
		return nil
	default:
		return fmt.Errorf("bun: can't scan %#v into NullTime", src)
	}
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
