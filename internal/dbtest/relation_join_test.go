package dbtest_test

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
	"testing"
)

type TestRelProfile struct {
	ID     int64 `bun:",pk,autoincrement"`
	Lang   string
	UserID int64
}

type TestRelUser struct {
	ID      int64 `bun:",pk,autoincrement"`
	Name    string
	Profile *TestRelProfile `bun:"rel:has-one,join:id=user_id"`
	Disks   []TestRelDisk   `bun:"rel:has-many,join:id=user_id"`
}

type TestRelDisk struct {
	ID     int64 `bun:",pk,autoincrement"`
	Title  string
	UserID int64
	User   *TestRelUser `bun:"rel:belongs-to,join:user_id=id"`
}

func TestRelationJoin(t *testing.T) {

	ctx := context.Background()

	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()

	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// Create schema

	models := []interface{}{
		(*TestRelUser)(nil),
		(*TestRelProfile)(nil),
		(*TestRelDisk)(nil),
	}
	for _, model := range models {
		_, err = db.NewCreateTable().Model(model).Exec(ctx)
		require.NoError(t, err)
	}

	expectedUsers := []*TestRelUser{
		{ID: 1, Name: "user 1"},
		{ID: 2, Name: "user 2"},
	}

	_, err = db.NewInsert().Model(&expectedUsers).Exec(ctx)
	require.NoError(t, err)

	expectedProfiles := []*TestRelProfile{
		{ID: 1, Lang: "en", UserID: 1},
		{ID: 2, Lang: "ru", UserID: 2},
	}

	_, err = db.NewInsert().Model(&expectedProfiles).Exec(ctx)
	require.NoError(t, err)

	expectedDisks := []*TestRelDisk{
		{ID: 1, Title: "Nirvana", UserID: 1},
		{ID: 2, Title: "Linkin Park", UserID: 2},
	}

	_, err = db.NewInsert().Model(&expectedDisks).Exec(ctx)
	require.NoError(t, err)

	// test Has One relation

	var users []TestRelUser
	err = db.NewSelect().
		Model(&users).
		Relation("Profile").
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, len(expectedUsers), len(users))

	// test Has One relation with filter

	users = []TestRelUser{}
	err = db.NewSelect().
		Model(&users).
		Relation("Profile", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("?TableAlias.lang = ?", "ru")
		}).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(users))

	// test Has One relation with join on

	users = []TestRelUser{}
	err = db.NewSelect().
		Model(&users).
		Relation("Profile", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.JoinOn("?TableAlias.lang = ?", "ru")
		}).
		OrderExpr("?TableAlias.ID").
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(users))
	require.Nil(t, users[0].Profile)
	require.NotNil(t, users[1].Profile)
	require.Equal(t, int64(2), users[1].Profile.ID)

	// test Has Many relation

	users = []TestRelUser{}
	err = db.NewSelect().
		Model(&users).
		Relation("Disks", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("?TableAlias.title = ?", "Linkin Park")
		}).
		Order("id").
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(users[0].Disks))
	require.Equal(t, 1, len(users[1].Disks))
	require.Equal(t, "Linkin Park", users[1].Disks[0].Title)

	// test Belongs To relation

	var disks []TestRelDisk
	err = db.NewSelect().
		Model(&disks).
		Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("?TableAlias.name = ?", "user 2")
		}).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(disks))
	require.Equal(t, "Linkin Park", disks[0].Title)
}
