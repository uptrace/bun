package dbtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
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

			b, err := event.IQuery.AppendQuery(schema.NewNopQueryGen(), nil)
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

// RawQuery used to render its comment in String() only and execute the
// query without it.
func TestRawQueryComment(t *testing.T) {
	testEachDB(t, testRawQueryComment)
}

func testRawQueryComment(t *testing.T, dbName string, db *bun.DB) {
	var sent []string
	db = db.WithQueryHook(&queryHook{
		beforeQuery: func(ctx context.Context, event *bun.QueryEvent) context.Context {
			sent = append(sent, event.Query)
			return ctx
		},
	})

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{"Comment with Scan", func() error {
			var num int
			return db.NewRaw("SELECT 1").Comment("raw").Scan(ctx, &num)
		}, "/* raw */ SELECT 1"},
		{"Comment with Exec", func() error {
			_, err := db.NewRaw("SELECT 1").Comment("raw").Exec(ctx)
			return err
		}, "/* raw */ SELECT 1"},
		{"comment from the context", func() error {
			var num int
			return db.NewRaw("SELECT 1").Scan(bun.WithComment(ctx, "raw"), &num)
		}, "/* raw */ SELECT 1"},
		{"no comment", func() error {
			var num int
			return db.NewRaw("SELECT ?", 1).Scan(ctx, &num)
		}, "SELECT 1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sent = sent[:0]
			require.NoError(t, test.run())
			require.Equal(t, []string{test.want}, sent)
		})
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
