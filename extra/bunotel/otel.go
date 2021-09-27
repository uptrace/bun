package bunotel

import (
	"context"
	"database/sql"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var (
	tracer         = otel.Tracer("github.com/uptrace/bun")
	meter          = metric.Must(global.Meter("github.com/uptrace/bun"))
	queryHistogram = meter.NewInt64Histogram(
		"bun.query.timing",
		metric.WithDescription("Timing of processed queries"),
		metric.WithUnit("milliseconds"),
	)
)

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
	operation := event.Operation()
	dbOperation := attribute.String("db.operation", operation)

	labels := []attribute.KeyValue{dbOperation}
	if event.IQuery != nil {
		if tableName := event.IQuery.GetTableName(); tableName != "" {
			labels = append(labels, attribute.String("db.table", tableName))
		}
	}

	queryHistogram.Record(ctx, time.Since(event.StartTime).Milliseconds(), labels...)

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetName(operation)
	defer span.End()

	query := eventQuery(event)
	fn, file, line := funcFileLine("github.com/uptrace/bun")

	attrs := make([]attribute.KeyValue, 0, 10)
	attrs = append(attrs,
		dbOperation,
		attribute.String("db.statement", query),
		attribute.String("code.function", fn),
		attribute.String("code.filepath", file),
		attribute.Int("code.lineno", line),
	)

	if s := dbSystem(event.DB); s != "" {
		attrs = append(attrs, attribute.String("db.system", s))
	}
	if event.Result != nil {
		if n, _ := event.Result.RowsAffected(); n > 0 {
			attrs = append(attrs, attribute.Int64("db.rows_affected", n))
		}
	}

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

func eventQuery(event *bun.QueryEvent) string {
	const softQueryLimit = 8000
	const hardQueryLimit = 16000

	var query string

	if len(event.Query) > softQueryLimit {
		query = unformattedQuery(event)
	} else {
		query = event.Query
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
		return ""
	}
}
