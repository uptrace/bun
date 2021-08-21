package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

type User struct {
	ID        int64
	Name      sql.NullString
	Email     string
	CreatedAt time.Time
	UpdatedAt sql.NullTime
}

type Org struct {
	ID      int64
	Name    string
	OwnerID int64
	Owner   *User `bun:"rel:belongs-to"`
}

func main() {
	ctx := context.Background()

	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

	db.RegisterModel((*User)(nil), (*Org)(nil))

	fixture := dbfixture.New(db, dbfixture.WithRecreateTables())
	if err := fixture.Load(ctx, os.DirFS("."), "fixture.yaml"); err != nil {
		panic(err)
	}

	fmt.Println("Smith", fixture.MustRow("User.smith").(*User))
	fmt.Println("Org with id=1", fixture.MustRow("Org.pk1").(*Org))
}
