package bun

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun/schema"
)

type IQuery interface {
	schema.QueryAppender
	Operation() string
	GetModel() Model
	GetTableName() string
}

type QueryEvent struct {
	DB *DB

	QueryAppender schema.QueryAppender // Deprecated: use IQuery instead
	IQuery        IQuery
	Query         string
	QueryArgs     []interface{}
	Model         Model

	StartTime time.Time
	Result    sql.Result
	Err       error

	Stash map[interface{}]interface{}
}

func (e *QueryEvent) Operation() string {
	if e.IQuery != nil {
		return e.IQuery.Operation()
	}
	return queryOperation(e.Query)
}

func queryOperation(query string) string {
	if idx := strings.IndexByte(query, ' '); idx > 0 {
		query = query[:idx]
	}
	if len(query) > 16 {
		query = query[:16]
	}
	return query
}

type QueryHook interface {
	BeforeQuery(context.Context, *QueryEvent) context.Context
	AfterQuery(context.Context, *QueryEvent)
}

func (db *DB) beforeQuery(
	ctx context.Context,
	iquery IQuery,
	query string,
	queryArgs []interface{},
	model Model,
) (context.Context, *QueryEvent) {
	atomic.AddUint32(&db.stats.Queries, 1)

	if len(db.queryHooks) == 0 {
		return ctx, nil
	}

	event := &QueryEvent{
		DB: db,

		Model:         model,
		QueryAppender: iquery,
		IQuery:        iquery,
		Query:         query,
		QueryArgs:     queryArgs,

		StartTime: time.Now(),
	}

	for _, hook := range db.queryHooks {
		ctx = hook.BeforeQuery(ctx, event)
	}

	return ctx, event
}

func (db *DB) afterQuery(
	ctx context.Context,
	event *QueryEvent,
	res sql.Result,
	err error,
) {
	switch err {
	case nil, sql.ErrNoRows:
		// nothing
	default:
		atomic.AddUint32(&db.stats.Errors, 1)
	}

	if event == nil {
		return
	}

	event.Result = res
	event.Err = err

	db.afterQueryFromIndex(ctx, event, len(db.queryHooks)-1)
}

func (db *DB) afterQueryFromIndex(ctx context.Context, event *QueryEvent, hookIndex int) {
	for ; hookIndex >= 0; hookIndex-- {
		db.queryHooks[hookIndex].AfterQuery(ctx, event)
	}
}

//------------------------------------------------------------------------------

func callBeforeScanHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.BeforeScanHook).BeforeScan(ctx)
}

func callAfterScanHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.AfterScanHook).AfterScan(ctx)
}
