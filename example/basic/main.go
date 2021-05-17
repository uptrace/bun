package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-mysql-org/go-mysql/driver"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"
)

type User struct {
	ID     int64 `bun:",autoincrement"`
	Name   string
	Emails []string
}

func (u User) String() string {
	return fmt.Sprintf("User<%d %s %v>", u.ID, u.Name, u.Emails)
}

type Story struct {
	ID       int64 `bun:",autoincrement"`
	Title    string
	AuthorID int64
	Author   *User `bun:"rel:has-one"`
}

func main() {
	ctx := context.Background()

	sqlite, err := sql.Open("sqlite3", ":memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqlite.SetMaxOpenConns(1)

	db := bun.NewDB(sqlite, sqlitedialect.New())
	if false {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))
	}

	// Drop and create tables.
	models := []interface{}{
		(*User)(nil),
		(*Story)(nil),
	}
	for _, model := range models {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		if err != nil {
			panic(err)
		}

		_, err = db.NewCreateTable().Model(model).Exec(ctx)
		if err != nil {
			panic(err)
		}
	}

	// Bulk-insert multiple users.
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
	_, err = db.NewInsert().Model(&users).Exec(ctx)
	if err != nil {
		panic(err)
	}

	// Insert one story.
	story1 := &Story{
		Title:    "Cool story",
		AuthorID: users[0].ID,
	}
	_, err = db.NewInsert().Model(story1).Exec(ctx)
	if err != nil {
		panic(err)
	}

	// Select all users.
	users = make([]User, 0)
	err = db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("users", users)

	// Select one user by primary key.
	user1 := new(User)
	err = db.NewSelect().Model(user1).Where("id = ?", 1).Scan(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("user1", user1)

	// Select the story and the associated author in a single query.
	story := new(Story)
	err = db.NewSelect().
		Model(story).
		Relation("Author").
		Limit(1).
		Scan(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(story)
}
