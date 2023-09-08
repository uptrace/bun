package bunzerolog

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/uptrace/bun"
)

var _ bun.QueryHook = (*QueryHook)(nil)

type QueryHook struct{}

// BeforeQuery before query zerolog hook.
func (h *QueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery after query zerolog hook.
func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	l := log.Ctx(ctx).With().
		Str("query", event.Query).
		Str("operation", event.Operation()).
		Str("duration", time.Since(event.StartTime).String()).
		Logger()

	if event.Err != nil {
		// do not log sql.ErrNoRows as real error
		if errors.Is(event.Err, sql.ErrNoRows) {
			l.Warn().Err(event.Err).Send()
			return
		}

		l.Err(event.Err).Send()
		return
	}

	l.Debug().Send()
}
