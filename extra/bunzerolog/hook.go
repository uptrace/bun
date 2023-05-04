package bunzerolog

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/TommyLeng/bun"
	"github.com/rs/zerolog"
)

var _ bun.QueryHook = (*QueryHook)(nil)

type QueryHook struct{}

// BeforeQuery before query zerolog hook.
func (h *QueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery after query zerolog hook.
func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	var logEvent *zerolog.Event

	// do not log sql.ErrNoRows as real error
	l := zerolog.Ctx(ctx)
	if errors.Is(event.Err, sql.ErrNoRows) {
		logEvent = l.Warn().Err(event.Err)
	} else {
		logEvent = l.Err(event.Err)
	}

	logEvent.
		Str("query", event.Query).
		Str("operation", event.Operation()).
		Str("duration", time.Since(event.StartTime).String()).
		Msg("query")
}
