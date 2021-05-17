package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/example/migrate/migrations"
	"github.com/uptrace/bun/extra/bundebug"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/urfave/cli/v2"
)

func main() {
	sqldb, err := sql.Open("sqlite3", "file:test.s3db?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.Open(sqldb, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

	migrations := migrations.Migrations

	app := &cli.App{
		Name: "bun",

		Commands: []*cli.Command{
			{
				Name:  "db",
				Usage: "database migrations",
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "create migration tables",
						Action: func(c *cli.Context) error {
							return migrations.Init(c.Context, db)
						},
					},
					{
						Name:  "migrate",
						Usage: "migrate database",
						Action: func(c *cli.Context) error {
							return migrations.Migrate(c.Context, db)
						},
					},
					{
						Name:  "rollback",
						Usage: "rollback the last migration batch",
						Action: func(c *cli.Context) error {
							return migrations.Rollback(c.Context, db)
						},
					},
					{
						Name:  "lock",
						Usage: "lock migrations",
						Action: func(c *cli.Context) error {
							return migrations.Lock(c.Context, db)
						},
					},
					{
						Name:  "unlock",
						Usage: "unlock migrations",
						Action: func(c *cli.Context) error {
							return migrations.Unlock(c.Context, db)
						},
					},
					{
						Name:  "create_go",
						Usage: "create Go migration",
						Action: func(c *cli.Context) error {
							return migrations.CreateGo(c.Context, db, c.Args().Get(0))
						},
					},
					{
						Name:  "create_sql",
						Usage: "create SQL migration",
						Action: func(c *cli.Context) error {
							return migrations.CreateSQL(c.Context, db, c.Args().Get(0))
						},
					},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
