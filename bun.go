package bun

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/sqlfmt"
)

type (
	Safe  = sqlfmt.Safe
	Ident = sqlfmt.Ident
)

type BaseModel struct{}

func In(slice interface{}) sqlfmt.InValues {
	return sqlfmt.In(slice)
}

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
	_ sqlfmt.QueryAppender = (*NullTime)(nil)
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

func (tm NullTime) AppendQuery(fmter sqlfmt.Formatter, b []byte) ([]byte, error) {
	if tm.IsZero() {
		return sqlfmt.AppendNull(b), nil
	}
	return sqlfmt.AppendTime(b, tm.Time), nil
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
