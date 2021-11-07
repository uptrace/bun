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
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var (
	tracer = otel.Tracer("github.com/uptrace/bun")
	meter  = metric.Must(global.Meter("github.com/uptrace/bun"))

	queryHistogram = meter.NewInt64Histogram(
		"go.sql.query_timing",
		metric.WithDescription("Timing of processed queries"),
		metric.WithUnit("milliseconds"),
	)
)

type QueryHook struct {
	attrs []attribute.KeyValue
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

	reportDBStats(db.DB, labels)
}

func (h *QueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	ctx, _ = tracer.Start(ctx, "")
	return ctx
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	operation := event.Operation()
	dbOperation := semconv.DBOperationKey.String(operation)

	labels := make([]attribute.KeyValue, 0, 2)
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

	query := eventQuery(event)
	fn, file, line := funcFileLine("github.com/uptrace/bun")

	attrs := make([]attribute.KeyValue, 0, 10)
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

func dbSystem(db *bun.DB) attribute.KeyValue {
	switch db.Dialect().Name() {
	case dialect.PG:
		return semconv.DBSystemPostgreSQL
	case dialect.MySQL:
		return semconv.DBSystemMySQL
	case dialect.SQLite:
		return semconv.DBSystemSqlite
	default:
		return attribute.KeyValue{}
	}
}

func reportDBStats(db *sql.DB, labels []attribute.KeyValue) {
	var maxOpenConns metric.Int64GaugeObserver
	var openConns metric.Int64GaugeObserver
	var inUseConns metric.Int64GaugeObserver
	var idleConns metric.Int64GaugeObserver
	var connsWaitCount metric.Int64CounterObserver
	var connsWaitDuration metric.Int64CounterObserver
	var connsClosedMaxIdle metric.Int64CounterObserver
	var connsClosedMaxIdleTime metric.Int64CounterObserver
	var connsClosedMaxLifetime metric.Int64CounterObserver

	batch := meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		stats := db.Stats()

		result.Observe(labels,
			maxOpenConns.Observation(int64(stats.MaxOpenConnections)),

			openConns.Observation(int64(stats.OpenConnections)),
			inUseConns.Observation(int64(stats.InUse)),
			idleConns.Observation(int64(stats.Idle)),

			connsWaitCount.Observation(stats.WaitCount),
			connsWaitDuration.Observation(int64(stats.WaitDuration)),
			connsClosedMaxIdle.Observation(stats.MaxIdleClosed),
			connsClosedMaxIdleTime.Observation(stats.MaxIdleTimeClosed),
			connsClosedMaxLifetime.Observation(stats.MaxLifetimeClosed),
		)
	})

	maxOpenConns = batch.NewInt64GaugeObserver("go.sql.connections_max_open",
		metric.WithDescription("Maximum number of open connections to the database"),
	)
	openConns = batch.NewInt64GaugeObserver("go.sql.connections_open",
		metric.WithDescription("The number of established connections both in use and idle"),
	)
	inUseConns = batch.NewInt64GaugeObserver("go.sql.connections_in_use",
		metric.WithDescription("The number of connections currently in use"),
	)
	idleConns = batch.NewInt64GaugeObserver("go.sql.connections_idle",
		metric.WithDescription("The number of idle connections"),
	)
	connsWaitCount = batch.NewInt64CounterObserver("go.sql.connections_wait_count",
		metric.WithDescription("The total number of connections waited for"),
	)
	connsWaitDuration = batch.NewInt64CounterObserver("go.sql.connections_wait_duration",
		metric.WithDescription("The total time blocked waiting for a new connection"),
		metric.WithUnit("nanoseconds"),
	)
	connsClosedMaxIdle = batch.NewInt64CounterObserver("go.sql.connections_closed_max_idle",
		metric.WithDescription("The total number of connections closed due to SetMaxIdleConns"),
	)
	connsClosedMaxIdleTime = batch.NewInt64CounterObserver("go.sql.connections_closed_max_idle_time",
		metric.WithDescription("The total number of connections closed due to SetConnMaxIdleTime"),
	)
	connsClosedMaxLifetime = batch.NewInt64CounterObserver("go.sql.connections_closed_max_lifetime",
		metric.WithDescription("The total number of connections closed due to SetConnMaxLifetime"),
	)
}
