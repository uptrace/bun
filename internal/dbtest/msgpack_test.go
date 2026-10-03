package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// TestMsgpackRoundTrip covers the reported symptom directly: a msgpack column is
// written as a byte literal, and each dialect spells those differently. Writing
// PostgreSQL's '\x..' form everywhere stored the value as text on the others,
// where it could no longer be decoded.
func TestMsgpackRoundTrip(t *testing.T) {
	type Item struct {
		Something int `msgpack:"something"`
	}

	type Model struct {
		bun.BaseModel `bun:"table:msgpack_models"`

		ID      int64 `bun:",pk,autoincrement"`
		Encoded Item  `bun:",msgpack"`
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		ctx := context.Background()
		mustResetModel(t, ctx, db, (*Model)(nil))

		inserted := &Model{Encoded: Item{Something: 1}}
		_, err := db.NewInsert().Model(inserted).Exec(ctx)
		require.NoError(t, err)

		selected := new(Model)
		err = db.NewSelect().Model(selected).Where("id = ?", inserted.ID).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, inserted.Encoded, selected.Encoded)
	})
}
