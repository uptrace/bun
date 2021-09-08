package dbtest_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

func TestSoftDelete(t *testing.T) {
	type Test struct {
		run func(t *testing.T, db *bun.DB)
	}

	tests := []Test{
		{run: testSoftDeleteAPI},
		{run: testSoftDeleteBulk},
	}
	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for _, test := range tests {
			t.Run(funcName(test.run), func(t *testing.T) {
				test.run(t, db)
			})
		}
	})
}

type CustomTime struct {
	Time time.Time
}

var _ driver.Valuer = (*CustomTime)(nil)

func (tm *CustomTime) Value() (driver.Value, error) {
	return tm.Time, nil
}

var _ sql.Scanner = (*CustomTime)(nil)

func (tm *CustomTime) Scan(src interface{}) error {
	tm.Time, _ = src.(time.Time)
	return nil
}

func (tm *CustomTime) IsZero() bool {
	return tm.Time.IsZero()
}

type Video struct {
	ID        int64
	Name      string
	DeletedAt time.Time `bun:",soft_delete"`
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

	_, err = db.NewDelete().Model(&videos).WherePK().Exec(ctx)
	require.NoError(t, err)

	count, err := db.NewSelect().Model((*Video)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
