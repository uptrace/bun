package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dbfixture"
	"github.com/TommyLeng/bun/dialect/sqlitedialect"
	"github.com/TommyLeng/bun/driver/sqliteshim"
	"github.com/TommyLeng/bun/extra/bundebug"
	"github.com/davecgh/go-spew/spew"
)

type Comment struct {
	TrackableID   int64  // Article.ID or Post.ID
	TrackableType string // "article" or "post"
	Text          string
}

type Article struct {
	ID   int64 `bun:",pk,autoincrement"`
	Name string

	Comments []Comment `bun:"rel:has-many,join:id=trackable_id,join:type=trackable_type,polymorphic"`
}

type Post struct {
	ID   int64 `bun:",pk,autoincrement"`
	Name string

	Comments []Comment `bun:"rel:has-many,join:id=trackable_id,join:type=trackable_type,polymorphic"`
}

func main() {
	ctx := context.Background()

	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()

	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	if err := createSchema(ctx, db); err != nil {
		panic(err)
	}

	{
		article := new(Article)
		if err := db.NewSelect().
			Model(article).
			Relation("Comments").
			Where("id = 1").
			Scan(ctx); err != nil {
			panic(err)
		}
		spew.Dump(article)
	}

	{
		post := new(Post)
		if err := db.NewSelect().
			Model(post).
			Relation("Comments").
			Where("id = 1").
			Scan(ctx); err != nil {
			panic(err)
		}
		spew.Dump(post)
	}
}

func createSchema(ctx context.Context, db *bun.DB) error {
	// Register models for the fixture.
	db.RegisterModel((*Comment)(nil), (*Article)(nil), (*Post)(nil))

	// Create tables and load initial data.
	fixture := dbfixture.New(db, dbfixture.WithRecreateTables())
	if err := fixture.Load(ctx, os.DirFS("."), "fixture.yml"); err != nil {
		return err
	}

	return nil
}
