package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

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

	// Select all users.
	users := make([]User, 0)
	if err := db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("all users: %v\n\n", users)

	// Select one user by primary key.
	user1 := new(User)
	if err := db.NewSelect().Model(user1).Where("id = ?", 1).Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("user1: %v\n\n", user1)

	// Select a story and the associated author in a single query.
	story := new(Story)
	if err := db.NewSelect().
		Model(story).
		Relation("Author").
		Limit(1).
		Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("story and the author: %v\n\n", story)

	// Select a user into a map.
	var m map[string]interface{}
	if err := db.NewSelect().
		Model((*User)(nil)).
		Limit(1).
		Scan(ctx, &m); err != nil {
		panic(err)
	}
	fmt.Printf("story map: %v\n\n", m)

	// Select all users scanning each column into a separate slice.
	var ids []int64
	var names []string
	if err := db.NewSelect().
		ColumnExpr("id, name").
		Model((*User)(nil)).
		OrderExpr("id ASC").
		Scan(ctx, &ids, &names); err != nil {
		panic(err)
	}
	fmt.Printf("users columns: %v %v\n\n", ids, names)
}

type User struct {
	ID     int64
	Name   string
	Emails []string
}

func (u User) String() string {
	return fmt.Sprintf("User<%d %s %v>", u.ID, u.Name, u.Emails)
}

type Story struct {
	ID       int64
	Title    string
	AuthorID int64
	Author   *User `bun:"rel:has-one"`
}
