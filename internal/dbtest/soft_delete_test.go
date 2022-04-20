package dbtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/feature"
)

func TestSoftDelete(t *testing.T) {
	type Test struct {
		run func(t *testing.T, db *bun.DB)
	}

	tests := []Test{
		{run: testSoftDeleteNilModel},
		{run: testSoftDeleteAPI},
		{run: testSoftDeleteBulk},
		{run: testSoftDeleteForce},
	}
	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for _, test := range tests {
			t.Run(funcName(test.run), func(t *testing.T) {
				test.run(t, db)
			})
		}
	})
}

type Video struct {
	ID        int64 `bun:",pk,autoincrement"`
	Name      string
	DeletedAt time.Time `bun:",soft_delete,nullzero"`
}

func testSoftDeleteNilModel(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	err := db.ResetModel(ctx, (*Video)(nil))
	require.NoError(t, err)

	_, err = db.NewDelete().Model((*Video)(nil)).Where("1 = 1").Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewDelete().Model((*Video)(nil)).Where("1 = 1").ForceDelete().Exec(ctx)
	require.NoError(t, err)
}

func testSoftDeleteAPI(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	err := db.ResetModel(ctx, (*Video)(nil))
	require.NoError(t, err)

	video1 := &Video{
		ID: 1,
	}
	_, err = db.NewInsert().Model(video1).Exec(ctx)
	require.NoError(t, err)

	// Count visible videos.
	count, err := db.NewSelect().Model((*Video)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Soft delete.
	_, err = db.NewDelete().Model(video1).Where("id = ?", video1.ID).Exec(ctx)
	require.NoError(t, err)

	// Count visible videos.
	count, err = db.NewSelect().Model((*Video)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Count deleted videos.
	count, err = db.NewSelect().Model((*Video)(nil)).WhereDeleted().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Undelete.
	_, err = db.NewUpdate().
		Model(video1).
		Set("deleted_at = NULL").
		WherePK().
		WhereAllWithDeleted().
		Exec(ctx)
	require.NoError(t, err)

	// Count visible videos.
	count, err = db.NewSelect().Model((*Video)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Force delete.
	_, err = db.NewDelete().Model(video1).Where("id = ?", video1.ID).ForceDelete().Exec(ctx)
	require.NoError(t, err)

	// Count deleted.
	count, err = db.NewSelect().Model((*Video)(nil)).WhereDeleted().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testSoftDeleteForce(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	err := db.ResetModel(ctx, (*Video)(nil))
	require.NoError(t, err)

	video1 := &Video{
		ID: 1,
	}
	_, err = db.NewInsert().Model(video1).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewDelete().Model(video1).WherePK().ForceDelete().Exec(ctx)
	require.NoError(t, err)

	count, err := db.NewSelect().Model((*Video)(nil)).WhereAllWithDeleted().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testSoftDeleteBulk(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	err := db.ResetModel(ctx, (*Video)(nil))
	require.NoError(t, err)

	videos := []Video{
		{Name: "video1"},
		{Name: "video2"},
	}
	_, err = db.NewInsert().Model(&videos).Exec(ctx)
	require.NoError(t, err)

	if db.Dialect().Features().Has(feature.CTE) {
		_, err := db.NewUpdate().
			Model(&videos).
			Column("name").
			Bulk().
			Exec(ctx)
		require.NoError(t, err)
	}

	_, err = db.NewDelete().Model(&videos).WherePK().Exec(ctx)
	require.NoError(t, err)

	count, err := db.NewSelect().Model((*Video)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
