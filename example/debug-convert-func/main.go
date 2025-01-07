package main

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
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
		bundebug.WithQueryConvertFunc(func(query string) string {
			// truncate long query values

			maxLength := 20

			return regexp.MustCompile(`'(?:[^'\\]|\\.)*'`).ReplaceAllStringFunc(query, func(match string) string {
				content := match[1 : len(match)-1]
				if len(content) > maxLength {
					return "'" + content[:maxLength] + "... [TRUNCATED]'"
				}
				return match
			})
		}),
	))

	if err := resetSchema(ctx, db); err != nil {
		panic(err)
	}

	users := make([]User, 0)
	// log `WHERE (name = abcabcabcabcabcabcab... [TRUNCATED]')`
	if err := db.NewSelect().Model(&users).Where("name = ?", strings.Repeat("abc", 100)).Scan(ctx); err != nil {
		panic(err)
	}
}

type User struct {
	ID     int64 `bun:",pk,autoincrement"`
	Name   string
	Emails []string
}

func resetSchema(ctx context.Context, db *bun.DB) error {
	if err := db.ResetModel(ctx, (*User)(nil)); err != nil {
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

	return nil
}
