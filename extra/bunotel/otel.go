package bunotel

import (
	"bytes"
	"context"
	"database/sql"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var tracer = otel.Tracer("github.com/uptrace/bun")

type ConfigOption func(*QueryHook)

type QueryHook struct{}

var _ bun.QueryHook = (*QueryHook)(nil)

func NewQueryHook(opts ...ConfigOption) *QueryHook {
	h := new(QueryHook)
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *QueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	ctx, _ = tracer.Start(ctx, "")
	return ctx
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	defer span.End()

	operation := eventOperation(event)
	query := eventQuery(event, operation)
	fn, file, line := funcFileLine("github.com/uptrace/bun")

	attrs := make([]attribute.KeyValue, 0, 10)
	attrs = append(attrs,
		attribute.String("db.system", dbSystem(event.DB)),
		attribute.String("db.statement", query),

		attribute.String("code.function", fn),
		attribute.String("code.filepath", file),
		attribute.Int("code.lineno", line),
	)

	if event.Err != nil {
		switch event.Err {
		case sql.ErrNoRows:
		default:
			span.RecordError(event.Err)
			span.SetStatus(codes.Error, event.Err.Error())
		}
	}

	span.SetAttributes(attrs...)
}

func funcFileLine(pkg string) (string, string, int) {
	const depth = 16
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	ff := runtime.CallersFrames(pcs[:n])

	var fn, file string
	var line int
	for {
		f, ok := ff.Next()
		if !ok {
			break
		}
		fn, file, line = f.Function, f.File, f.Line
		if !strings.Contains(fn, pkg) {
			break
		}
	}

	if ind := strings.LastIndexByte(fn, '/'); ind != -1 {
		fn = fn[ind+1:]
	}

	return fn, file, line
}

func eventOperation(event *bun.QueryEvent) string {
	switch event.QueryAppender.(type) {
	case *bun.SelectQuery:
		return "SELECT"
	case *bun.InsertQuery:
		return "INSERT"
	case *bun.UpdateQuery:
		return "UPDATE"
	case *bun.DeleteQuery:
		return "DELETE"
	case *bun.CreateTableQuery:
		return "CREATE TABLE"
	case *bun.DropTableQuery:
		return "DROP TABLE"
	}
	return queryOperation(event.Query)
}

func queryOperation(name []byte) string {
	if idx := bytes.IndexByte(name, ' '); idx > 0 {
		name = name[:idx]
	}
	if len(name) > 16 {
		name = name[:16]
	}
	return string(name)
}

func eventQuery(event *bun.QueryEvent, operation string) string {
	const softQueryLimit = 5000
	const hardQueryLimit = 10000

	var query string

	if len(event.Query) > softQueryLimit {
		query = unformattedQuery(event)
	} else {
		query = string(event.Query)
	}

	if len(query) > hardQueryLimit {
		query = query[:hardQueryLimit]
	}

	return query
}

func unformattedQuery(event *bun.QueryEvent) string {
	if b, err := event.QueryAppender.AppendQuery(schema.NewNopFormatter(), nil); err == nil {
		return bytesToString(b)
	}
	return string(event.Query)
}

func dbSystem(db *bun.DB) string {
	switch db.Dialect().Name() {
	case dialect.PG:
		return "postgresql"
	case dialect.MySQL:
		return "mysql"
	case dialect.SQLite:
		return "sqlite"
	default:
		return "unknown"
	}
}
