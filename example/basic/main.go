package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/sqlitedialect"
	"github.com/TommyLeng/bun/driver/sqliteshim"
	"github.com/TommyLeng/bun/extra/bundebug"
)

func main() {
	ctx := context.Background()

	sqlite, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqlite.SetMaxOpenConns(1)

	db := bun.NewDB(sqlite, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	if err := resetSchema(ctx, db); err != nil {
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
	fmt.Printf("user map: %v\n\n", m)

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
	ID     int64 `bun:",pk,autoincrement"`
	Name   string
	Emails []string
}

func (u User) String() string {
	return fmt.Sprintf("User<%d %s %v>", u.ID, u.Name, u.Emails)
}

type Story struct {
	ID       int64 `bun:",pk,autoincrement"`
	Title    string
	AuthorID int64
	Author   *User `bun:"rel:belongs-to,join:author_id=id"`
}

func resetSchema(ctx context.Context, db *bun.DB) error {
	if err := db.ResetModel(ctx, (*User)(nil), (*Story)(nil)); err != nil {
		return err
	}

	users := []User{
		{
			Name:   "admin",
			Emails: []string{"admin1@admin", "admin2@admin"},
		},
		{
			Name:   "root",
			Emails: []string{"root1@root", "root2@root"},
		},
	}
	if _, err := db.NewInsert().Model(&users).Exec(ctx); err != nil {
		return err
	}

	stories := []Story{
		{
			Title:    "Cool story",
			AuthorID: users[0].ID,
		},
	}
	if _, err := db.NewInsert().Model(&stories).Exec(ctx); err != nil {
		return err
	}

	return nil
}
