package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"
)

func main() {
	ctx := context.Background()

	sqlite, err := sql.Open("sqlite3", ":memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqlite.SetMaxOpenConns(1)

	db := bun.NewDB(sqlite, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

	// Register models for the fixture.
	db.RegisterModel((*User)(nil), (*Story)(nil))

	// Create tables and load initial data.
	fixture := dbfixture.New(db, dbfixture.WithRecreateTables())
	if err := fixture.Load(ctx, os.DirFS("."), "fixture.yaml"); err != nil {
		panic(err)
	}

	stories := make([]*Story, 0)
	if err := db.NewSelect().Model(&stories).Scan(ctx); err != nil {
		panic(err)
	}

	spew.Dump(stories)
}

type User struct {
	ID     int64
	Name   string
	Emails []string
}

func SelectUser(ctx context.Context, db *bun.DB, userID int64) (*User, error) {
	user := new(User)
	if err := db.NewSelect().Model(user).Where("id = ?", userID).Scan(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

type Story struct {
	ID       int64
	Title    string
	AuthorID int64
	Author   *User `bun:"rel:has-one"`
}

var _ bun.AfterSelectHook = (*Story)(nil)

func (s *Story) AfterSelect(ctx context.Context, query *bun.SelectQuery) error {
	db := query.DB()
	value := query.GetModel().Value()

	stories, ok := value.(*[]*Story)
	if !ok {
		return fmt.Errorf("Story.AfterSelect: unexpected %T", value)
	}

	for _, story := range *stories {
		author, err := SelectUser(ctx, db, story.AuthorID)
		if err != nil {
			return err
		}
		story.Author = author
	}

	return nil
}
