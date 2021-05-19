package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/fixture"
)

type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
}

type Org struct {
	ID      int64
	Name    string
	OwnerID int64
	Owner   *User `bun:"rel:has-one"`
}

func main() {
	ctx := context.Background()

	sqldb, err := sql.Open("sqlite3", ":memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

	db.RegisterModel((*User)(nil), (*Org)(nil))

	loader := fixture.NewLoader(db, fixture.WithRecreateTables())
	if err := loader.Load(ctx, os.DirFS("."), "fixture.yaml"); err != nil {
		panic(err)
	}

	fmt.Println("Smith", loader.MustRow("User.smith").(*User))
	fmt.Println("Org with id=1", loader.MustRow("Org.pk1").(*Org))
}
