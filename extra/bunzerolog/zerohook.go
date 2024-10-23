package bunzerolog

// bunslog provides logging functionalities for Bun using slog.
// This package allows SQL queries issued by Bun to be displayed using slog.

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/uptrace/bun"
)

var _ bun.QueryHook = (*QueryHook)(nil)

// Option is a function that configures a QueryHook.
type Option func(*QueryHook)

// WithLogger sets the *zerolog.Logger instance.
func WithLogger(logger *zerolog.Logger) Option {
	return func(h *QueryHook) {
		h.logger = logger
	}
}

// WithQueryLogLevel sets the log level for general queries.
func WithQueryLogLevel(level zerolog.Level) Option {
	return func(h *QueryHook) {
		h.queryLogLevel = level
	}
}

// WithSlowQueryLogLevel sets the log level for slow queries.
func WithSlowQueryLogLevel(level zerolog.Level) Option {
	return func(h *QueryHook) {
		h.slowQueryLogLevel = level
	}
}

// WithErrorQueryLogLevel sets the log level for queries that result in an error.
func WithErrorQueryLogLevel(level zerolog.Level) Option {
	return func(h *QueryHook) {
		h.errorLogLevel = level
	}
}

// WithSlowQueryThreshold sets the duration threshold for identifying slow queries.
func WithSlowQueryThreshold(threshold time.Duration) Option {
	return func(h *QueryHook) {
		h.slowQueryThreshold = threshold
	}
}

// WithLogFormat sets the custom format for slog output.
func WithLogFormat(f LogFormatFn) Option {
	return func(h *QueryHook) {
		h.logFormat = f
	}
}

type LogFormatFn func(ctx context.Context, event *bun.QueryEvent, zeroctx *zerolog.Event) *zerolog.Event

// QueryHook is a hook for Bun that enables logging with slog.
// It implements bun.QueryHook interface.
type QueryHook struct {
	logger             *zerolog.Logger
	queryLogLevel      zerolog.Level
	slowQueryLogLevel  zerolog.Level
	errorLogLevel      zerolog.Level
	slowQueryThreshold time.Duration
	logFormat          LogFormatFn
	now                func() time.Time
}

// NewQueryHook initializes a new QueryHook with the given options.
func NewQueryHook(opts ...Option) *QueryHook {
	h := &QueryHook{
		queryLogLevel:     zerolog.DebugLevel,
		slowQueryLogLevel: zerolog.WarnLevel,
		errorLogLevel:     zerolog.ErrorLevel,
		now:               time.Now,
	}

	for _, opt := range opts {
		opt(h)
	}

	// use default format
	if h.logFormat == nil {
		h.logFormat = func(ctx context.Context, event *bun.QueryEvent, zerevent *zerolog.Event) *zerolog.Event {
			duration := h.now().Sub(event.StartTime)

			return zerevent.
				Ctx(ctx).
				Err(event.Err).
				Str("query", event.Query).
				Str("operation", event.Operation()).
				Str("duration", duration.String())
		}
	}

	return h
}

// BeforeQuery is called before a query is executed.
func (h *QueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery is called after a query is executed.
// It logs the query based on its duration and whether it resulted in an error.
func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	level := h.queryLogLevel
	duration := h.now().Sub(event.StartTime)
	if h.slowQueryThreshold > 0 && h.slowQueryThreshold <= duration {
		level = h.slowQueryLogLevel
	}

	if event.Err != nil && !errors.Is(event.Err, sql.ErrNoRows) {
		level = h.errorLogLevel
	}

	l := h.logger
	if l == nil {
		l = log.Ctx(ctx)
	}

	h.logFormat(ctx, event, l.WithLevel(level)).Send()
}
