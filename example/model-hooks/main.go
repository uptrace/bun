package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/sqlitedialect"
	"github.com/TommyLeng/bun/driver/sqliteshim"
	"github.com/TommyLeng/bun/extra/bundebug"
	"github.com/davecgh/go-spew/spew"
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

	if err := db.ResetModel(ctx, (*User)(nil)); err != nil {
		panic(err)
	}

	user := new(User)

	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		panic(err)
	}

	if _, err := db.NewUpdate().
		Model(user).
		WherePK().
		Exec(ctx); err != nil {
		panic(err)
	}

	if err := db.NewSelect().Model(user).WherePK().Scan(ctx); err != nil {
		panic(err)
	}

	spew.Dump(user)
}

type User struct {
	ID        int64 `bun:",pk,autoincrement"`
	Password  string
	CreatedAt time.Time `bun:",nullzero"`
	UpdatedAt time.Time `bun:",nullzero"`
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.Password = "[encrypted]"
		u.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}

var _ bun.BeforeScanRowHook = (*User)(nil)

func (u *User) BeforeScanRow(ctx context.Context) error {
	// Do some initialization.
	u.Password = ""
	return nil
}

var _ bun.AfterScanRowHook = (*User)(nil)

func (u *User) AfterScanRow(ctx context.Context) error {
	u.Password = "[decrypted]"
	return nil
}
