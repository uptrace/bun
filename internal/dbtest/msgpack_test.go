package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

// TestMsgpackTag verifies the behaviour of the `,msgpack` struct tag per
// dialect. The tag encodes values as a PostgreSQL bytea hex literal, so it
// only works on PostgreSQL. On other dialects the value used to be stored
// verbatim as text and silently corrupted, failing on scan with a cryptic
// "msgpack: unexpected code=5c" error (#1219, #519). It must now fail fast at
// insert time with a clear, actionable error instead.
func TestMsgpackTag(t *testing.T) {
	type Item struct {
		Something int `msgpack:"something"`
	}

	type NonPGMsgpackModel struct {
		bun.BaseModel `bun:"table:msgpack_models"`

		ID      int64  `bun:",pk,autoincrement"`
		Name    string `bun:",notnull"`
		Encoded Item   `bun:",msgpack"`
	}

	t.Run("non-pg", func(t *testing.T) {
		t.Run("sqlite", func(t *testing.T) {
			db := sqlite(t)
			ctx := context.Background()
			mustResetModel(t, ctx, db, (*NonPGMsgpackModel)(nil))

			model := &NonPGMsgpackModel{Name: "test", Encoded: Item{Something: 1}}

			// Non-PostgreSQL: the insert must fail rather than silently storing a
			// corrupted value. Following bun's existing value-append error pattern
			// (see appendDriverValue), the readable reason is embedded in the
			// generated SQL via dialect.AppendError and the database rejects it.
			q := db.NewInsert().Model(model)
			require.Contains(t, q.String(),
				"msgpack struct tag is only supported by the PostgreSQL dialect")

			_, err := q.Exec(ctx)
			require.Error(t, err)
		})
	})

	t.Run("pg", func(t *testing.T) {
		testMsgpackTagPostgres(t, pg(t))
	})

	t.Run("pgx", func(t *testing.T) {
		testMsgpackTagPostgres(t, pgx(t))
	})
}

func testMsgpackTagPostgres(t *testing.T, db *bun.DB) {
	type Item struct {
		Something int `msgpack:"something"`
	}

	type MsgpackModel struct {
		bun.BaseModel `bun:"table:msgpack_models"`

		ID      int64  `bun:",pk,autoincrement"`
		Name    string `bun:",notnull"`
		Encoded Item   `bun:"type:bytea,msgpack"`
	}

	t.Helper()

	t.Run(db.String(), func(t *testing.T) {
		ctx := context.Background()
		mustResetModel(t, ctx, db, (*MsgpackModel)(nil))

		model := &MsgpackModel{Name: "test", Encoded: Item{Something: 1}}

		// Regression: msgpack must keep round-tripping on PostgreSQL.
		_, err := db.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)

		got := new(MsgpackModel)
		err = db.NewSelect().Model(got).Where("id = ?", model.ID).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, model.Encoded, got.Encoded)
	})
}
