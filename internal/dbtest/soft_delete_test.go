package dbtest_test

import (
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

func TestSoftDelete(t *testing.T) {
	testEachDB(t, testSoftDelete)
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

func testSoftDelete(t *testing.T, db *bun.DB) {
	_, err := db.NewDropTable().Model((*Video)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Model((*Video)(nil)).Exec(ctx)
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

	count, err = db.NewSelect().Model((*Video)(nil)).WhereDeleted().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = db.NewDelete().Model(video1).Where("id = ?", video1.ID).ForceDelete().Exec(ctx)
	require.NoError(t, err)

	count, err = db.NewSelect().Model((*Video)(nil)).WhereDeleted().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
