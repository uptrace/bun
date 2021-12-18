package bun

import (
	"context"
	"database/sql"
	"strings"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun/schema"
)

type QueryEvent struct {
	DB *DB

	QueryAppender schema.QueryAppender // Deprecated: use IQuery instead
	IQuery        Query
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
	query string,
	queryArgs []interface{},
) context.Context {
	atomic.AddUint32(&db.stats.Queries, 1)

	if len(db.queryHooks) == 0 {
		return ctx
	}

	scope := scopeFromContext(ctx)

	event := &QueryEvent{
		DB: db,

		Model:         scope.model,
		QueryAppender: scope.query,
		IQuery:        scope.query,
		Query:         query,
		QueryArgs:     queryArgs,

		StartTime: time.Now(),
	}

	ctx = withEvent(ctx, event)

	for _, hook := range db.queryHooks {
		ctx = hook.BeforeQuery(ctx, event)
	}

	return ctx
}

func (db *DB) afterQuery(
	ctx context.Context,
	res sql.Result,
	err error,
) {
	switch err {
	case nil, sql.ErrNoRows:
		// nothing
	default:
		atomic.AddUint32(&db.stats.Errors, 1)
	}

	event := scopeFromContext(ctx).event

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
