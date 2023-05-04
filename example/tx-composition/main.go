package main

import (
	"context"
	"database/sql"

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
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	if err := db.ResetModel(ctx, (*User)(nil), (*Profile)(nil)); err != nil {
		panic(err)
	}

	if err := insertUserAndProfile(ctx, db); err != nil {
		panic(err)
	}

	if err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return insertUserAndProfile(ctx, tx)
	}); err != nil {
		panic(err)
	}
}

func insertUserAndProfile(ctx context.Context, db bun.IDB) error {
	user := &User{
		Name: "Smith",
	}
	if err := InsertUser(ctx, db, user); err != nil {
		return err
	}

	profile := &Profile{
		UserID: user.ID,
		Email:  "iam@smith.com",
	}
	if err := InsertProfile(ctx, db, profile); err != nil {
		return err
	}

	return nil
}

type User struct {
	ID   int64 `bun:",pk,autoincrement"`
	Name string
}

func InsertUser(ctx context.Context, db bun.IDB, user *User) error {
	_, err := db.NewInsert().Model(user).Exec(ctx)
	return err
}

type Profile struct {
	ID     int64 `bun:",pk,autoincrement"`
	UserID int64
	Email  string
}

func InsertProfile(ctx context.Context, db bun.IDB, profile *Profile) error {
	_, err := db.NewInsert().Model(profile).Exec(ctx)
	return err
}
