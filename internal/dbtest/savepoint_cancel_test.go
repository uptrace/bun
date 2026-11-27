package dbtest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestCanceledSavepointRollsBack(t *testing.T) {
	type traceKey struct{}
	for _, tt := range []struct {
		name          string
		callbackError bool
	}{
		{"callback returns cancellation", true},
		{"callback returns nil", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, stop := context.WithTimeout(t.Context(), time.Minute)
			defer stop()
			deadline, _ := root.Deadline()
			var rolledBack bool
			db := sqlite(t).WithQueryHook(&queryHook{
				beforeQuery: func(ctx context.Context, event *bun.QueryEvent) context.Context {
					if strings.HasPrefix(event.Query, "ROLLBACK TO SAVEPOINT ") {
						rolledBack = true
						require.Equal(t, "child", ctx.Value(traceKey{}))
						got, ok := ctx.Deadline()
						require.True(t, ok)
						require.Equal(t, deadline, got)
						require.NoError(t, ctx.Err())
					}
					return ctx
				},
			})
			_, err := db.ExecContext(root, "CREATE TABLE records (id INTEGER)")
			require.NoError(t, err)

			err = db.RunInTx(root, nil, func(ctx context.Context, tx bun.Tx) error {
				parent, stop := context.WithTimeout(ctx, 30*time.Second)
				defer stop()
				return tx.RunInTx(parent, nil, func(ctx context.Context, sp bun.Tx) error {
					child, cancel := context.WithCancel(context.WithValue(ctx, traceKey{}, "child"))
					defer cancel()
					err := sp.RunInTx(child, nil, func(ctx context.Context, nested bun.Tx) error {
						if _, err := nested.ExecContext(ctx, "INSERT INTO records VALUES (1)"); err != nil {
							return err
						}
						cancel()
						if tt.callbackError {
							return ctx.Err()
						}
						return nil
					})
					require.ErrorIs(t, err, context.Canceled)
					_, err = sp.ExecContext(ctx, "INSERT INTO records VALUES (2)")
					return err
				})
			})
			require.NoError(t, err)
			require.True(t, rolledBack)

			var ids []int
			require.NoError(t, db.NewRaw("SELECT id FROM records").Scan(root, &ids))
			require.Equal(t, []int{2}, ids)
		})
	}
}
