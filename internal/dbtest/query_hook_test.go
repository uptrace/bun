package dbtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/schema"
)

func TestQueryHook(t *testing.T) {
	testEachDB(t, testQueryHook)
}

func testQueryHook(t *testing.T, dbName string, db *bun.DB) {
	hook := &queryHook{}
	db.AddQueryHook(hook)

	{
		hook.reset()
		hook.beforeQuery = func(
			ctx context.Context, event *bun.QueryEvent,
		) context.Context {
			require.Equal(t, "SELECT", event.Operation())
			require.Equal(
				t, "SELECT * FROM (SELECT 1 AS c) AS t WHERE (1 = 2)", string(event.Query))

			b, err := event.IQuery.AppendQuery(schema.NewNopFormatter(), nil)
			require.NoError(t, err)
			require.Equal(t, "SELECT * FROM (SELECT 1 AS c) AS t WHERE (? = ?)", string(b))

			return ctx
		}

		_, err := db.NewSelect().
			TableExpr("(SELECT 1 AS c) AS t").
			Where("? = ?", 1, 2).
			Exec(ctx)
		require.NoError(t, err)
		hook.require(t)
	}

	{
		hook.reset()
		hook.beforeQuery = func(
			ctx context.Context, event *bun.QueryEvent,
		) context.Context {
			require.Equal(t, "SELECT", event.Operation())
			require.Equal(t, "SELECT 1", string(event.Query))
			return ctx
		}

		_, err := db.Exec("SELECT 1")
		require.NoError(t, err)
		hook.require(t)
	}

	{
		hook.reset()
		hook.beforeQuery = func(
			ctx context.Context, event *bun.QueryEvent,
		) context.Context {
			require.Equal(t, "SELECT", event.Operation())
			require.Equal(t, "\n\t\t\tSELECT 1\n\t\t", string(event.Query))
			return ctx
		}

		var num int
		err := db.QueryRow(`
			SELECT 1
		`).Scan(&num)
		require.NoError(t, err)
		require.Equal(t, 1, num)
		hook.require(t)
	}
}

type queryHook struct {
	startTime time.Time
	endTime   time.Time

	beforeQuery func(context.Context, *bun.QueryEvent) context.Context
	afterQuery  func(context.Context, *bun.QueryEvent)
}

func (h *queryHook) BeforeQuery(
	ctx context.Context, evt *bun.QueryEvent,
) context.Context {
	h.startTime = time.Now()
	return h.beforeQuery(ctx, evt)
}

func (h *queryHook) AfterQuery(c context.Context, evt *bun.QueryEvent) {
	h.endTime = time.Now()
	if h.afterQuery != nil {
		h.afterQuery(ctx, evt)
	}
}

func (h *queryHook) reset() {
	*h = queryHook{}
}

func (h *queryHook) require(t *testing.T) {
	require.WithinDuration(t, h.startTime, time.Now(), time.Second)
	require.WithinDuration(t, h.endTime, time.Now(), time.Second)
}
