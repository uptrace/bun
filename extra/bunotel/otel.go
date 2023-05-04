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
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect"
	"github.com/TommyLeng/bun/schema"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
)

var (
	tracer = otel.Tracer("github.com/TommyLeng/bun")
	meter  = global.Meter("github.com/TommyLeng/bun")

	queryHistogram, _ = meter.Int64Histogram(
		"go.sql.query_timing",
		instrument.WithDescription("Timing of processed queries"),
		instrument.WithUnit("milliseconds"),
	)
)

type QueryHook struct {
	attrs         []attribute.KeyValue
	formatQueries bool
}

var _ bun.QueryHook = (*QueryHook)(nil)

func NewQueryHook(opts ...Option) *QueryHook {
	h := new(QueryHook)
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *QueryHook) Init(db *bun.DB) {
	labels := make([]attribute.KeyValue, 0, len(h.attrs)+1)
	labels = append(labels, h.attrs...)
	if sys := dbSystem(db); sys.Valid() {
		labels = append(labels, sys)
	}

	otelsql.ReportDBStatsMetrics(db.DB, otelsql.WithAttributes(labels...))
}

func (h *QueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	ctx, _ = tracer.Start(ctx, "", trace.WithSpanKind(trace.SpanKindClient))
	return ctx
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	operation := event.Operation()
	dbOperation := semconv.DBOperationKey.String(operation)

	labels := make([]attribute.KeyValue, 0, len(h.attrs)+2)
	labels = append(labels, h.attrs...)
	labels = append(labels, dbOperation)
	if event.IQuery != nil {
		if tableName := event.IQuery.GetTableName(); tableName != "" {
			labels = append(labels, semconv.DBSQLTableKey.String(tableName))
		}
	}

	queryHistogram.Record(ctx, time.Since(event.StartTime).Milliseconds(), labels...)

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetName(operation)
	defer span.End()

	query := h.eventQuery(event)
	fn, file, line := funcFileLine("github.com/TommyLeng/bun")

	attrs := make([]attribute.KeyValue, 0, 10)
	attrs = append(attrs, h.attrs...)
	attrs = append(attrs,
		dbOperation,
		semconv.DBStatementKey.String(query),
		semconv.CodeFunctionKey.String(fn),
		semconv.CodeFilepathKey.String(file),
		semconv.CodeLineNumberKey.Int(line),
	)

	if sys := dbSystem(event.DB); sys.Valid() {
		attrs = append(attrs, sys)
	}
	if event.Result != nil {
		if n, _ := event.Result.RowsAffected(); n > 0 {
			attrs = append(attrs, attribute.Int64("db.rows_affected", n))
		}
	}

	switch event.Err {
	case nil, sql.ErrNoRows, sql.ErrTxDone:
		// ignore
	default:
		span.RecordError(event.Err)
		span.SetStatus(codes.Error, event.Err.Error())
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

func (h *QueryHook) eventQuery(event *bun.QueryEvent) string {
	const softQueryLimit = 8000
	const hardQueryLimit = 16000

	var query string

	if h.formatQueries && len(event.Query) <= softQueryLimit {
		query = event.Query
	} else {
		query = unformattedQuery(event)
	}

	if len(query) > hardQueryLimit {
		query = query[:hardQueryLimit]
	}

	return query
}

func unformattedQuery(event *bun.QueryEvent) string {
	if event.IQuery != nil {
		if b, err := event.IQuery.AppendQuery(schema.NewNopFormatter(), nil); err == nil {
			return bytesToString(b)
		}
	}
	return string(event.QueryTemplate)
}

func dbSystem(db *bun.DB) attribute.KeyValue {
	switch db.Dialect().Name() {
	case dialect.PG:
		return semconv.DBSystemPostgreSQL
	case dialect.MySQL:
		return semconv.DBSystemMySQL
	case dialect.SQLite:
		return semconv.DBSystemSqlite
	case dialect.MSSQL:
		return semconv.DBSystemMSSQL
	default:
		return attribute.KeyValue{}
	}
}
