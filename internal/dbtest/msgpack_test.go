package dbtest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

type msgpackPayload struct {
	Something int    `msgpack:"something"`
	Name      string `msgpack:"name"`
}

type msgpackModel struct {
	bun.BaseModel `bun:"table:msgpack_models"`

	ID      int64          `bun:",pk,autoincrement"`
	Encoded msgpackPayload `bun:",msgpack"`
}

// Regression test for https://github.com/uptrace/bun/issues/1219: a field tagged
// with `bun:",msgpack"` must round-trip on every dialect, not just PostgreSQL.
// The value was always written using PostgreSQL's '\x...' binary literal, which
// SQLite and MySQL store verbatim as text and then fail to decode on read back.
func TestMsgpackRoundTrip(t *testing.T) {
	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		_, err := db.NewDropTable().Model((*msgpackModel)(nil)).IfExists().Exec(ctx)
		require.NoError(t, err)

		_, err = db.NewCreateTable().Model((*msgpackModel)(nil)).Exec(ctx)
		require.NoError(t, err)
		defer func() {
			_, err := db.NewDropTable().Model((*msgpackModel)(nil)).IfExists().Exec(ctx)
			require.NoError(t, err)
		}()

		in := &msgpackModel{Encoded: msgpackPayload{Something: 42, Name: "hello"}}
		_, err = db.NewInsert().Model(in).Exec(ctx)
		require.NoError(t, err)

		out := new(msgpackModel)
		err = db.NewSelect().Model(out).Where("id = ?", in.ID).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, in.Encoded, out.Encoded)
	})
}
