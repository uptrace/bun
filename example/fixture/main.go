package main

import (
	"bytes"
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
	ID        int64 `bun:",pk,autoincrement"`
	Name      sql.NullString
	Email     string
	Attrs     map[string]interface{} `bun:",nullzero"`
	CreatedAt time.Time
	UpdatedAt sql.NullTime
}

type Org struct {
	ID      int64 `bun:",pk,autoincrement"`
	Name    string
	OwnerID int64 `yaml:"owner_id"`
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
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// Register models before loading fixtures.
	db.RegisterModel((*User)(nil), (*Org)(nil))

	// Automatically create tables.
	fixture := dbfixture.New(db, dbfixture.WithRecreateTables())

	// Load fixtures.
	if err := fixture.Load(ctx, os.DirFS("."), "fixture.yml"); err != nil {
		panic(err)
	}

	// You can access fixtures by _id and by a primary key.
	fmt.Println("Smith", fixture.MustRow("User.smith").(*User))
	fmt.Println("Org with id=1", fixture.MustRow("Org.pk1").(*Org))

	// Load users and orgs from the database.
	var users []User
	var orgs []Org

	if err := db.NewSelect().Model(&users).OrderExpr("id").Scan(ctx); err != nil {
		panic(err)
	}
	if err := db.NewSelect().Model(&orgs).OrderExpr("id").Scan(ctx); err != nil {
		panic(err)
	}

	// And encode the loaded data back as YAML.

	var buf bytes.Buffer
	enc := dbfixture.NewEncoder(db, &buf)

	if err := enc.Encode(users, orgs); err != nil {
		panic(err)
	}

	fmt.Println("")
	fmt.Println(buf.String())
}
