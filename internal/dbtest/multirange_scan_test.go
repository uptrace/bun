package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanMultirangeAndStringArrays(t *testing.T) {
	type Period string
	db := pg(t)
	ctx := context.Background()

	const multirangeSQL = `SELECT '{[1,2),[3,4)}'::int4multirange AS periods`

	t.Run("multirange/plain strings", func(t *testing.T) {
		var got struct {
			Periods []string `bun:"type:int4multirange"`
		}
		require.NoError(t, db.NewRaw(multirangeSQL).Scan(ctx, &got))
		require.Equal(t, []string{"[1,2)", "[3,4)"}, got.Periods)
	})
	t.Run("multirange/named strings", func(t *testing.T) {
		var got struct {
			Periods []Period `bun:",multirange"`
		}
		require.NoError(t, db.NewRaw(multirangeSQL).Scan(ctx, &got))
		require.Equal(t, []Period{"[1,2)", "[3,4)"}, got.Periods)
	})
	t.Run("text array", func(t *testing.T) {
		var got struct {
			Periods []string `bun:"type:text[]"`
		}
		require.NoError(t, db.NewRaw(`SELECT ARRAY['(555)123-4567', '[1,2)']::text[] AS periods`).Scan(ctx, &got))
		require.Equal(t, []string{"(555)123-4567", "[1,2)"}, got.Periods)
	})
	t.Run("range array", func(t *testing.T) {
		var got struct {
			Periods []string `bun:"type:int4range[]"`
		}
		require.NoError(t, db.NewRaw(`SELECT ARRAY['[1,2)'::int4range, '[3,4)'::int4range] AS periods`).Scan(ctx, &got))
		require.Equal(t, []string{"[1,2)", "[3,4)"}, got.Periods)
	})
}
