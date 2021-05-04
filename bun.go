package bun

import (
	"context"

	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/sqlfmt"
)

type (
	Safe  = sqlfmt.Safe
	Ident = sqlfmt.Ident
)

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

type BeforeSelectQueryHook interface {
	BeforeSelectQuery(ctx context.Context, query *SelectQuery) error
}

type AfterSelectQueryHook interface {
	AfterSelectQuery(ctx context.Context, query *SelectQuery) error
}

type BeforeInsertQueryHook interface {
	BeforeInsertQuery(ctx context.Context, query *InsertQuery) error
}

type AfterInsertQueryHook interface {
	AfterInsertQuery(ctx context.Context, query *InsertQuery) error
}

type BeforeUpdateQueryHook interface {
	BeforeUpdateQuery(ctx context.Context, query *UpdateQuery) error
}

type AfterUpdateQueryHook interface {
	AfterUpdateQuery(ctx context.Context, query *UpdateQuery) error
}

type BeforeDeleteQueryHook interface {
	BeforeDeleteQuery(ctx context.Context, query *DeleteQuery) error
}

type AfterDeleteQueryHook interface {
	AfterDeleteQuery(ctx context.Context, query *DeleteQuery) error
}

type BaseTable struct{}
