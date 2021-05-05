package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/uptrace/bun/dialect/sqlitedialect"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

var migrator = migrate.NewMigrator(migrate.AutoDiscover())

func main() {
	sqldb, err := sql.Open("sqlite3", "file:test.s3db?cache=shared")
	if err != nil {
		panic(err)
	}
	db := bun.Open(sqldb, sqlitedialect.New())

	app := &cli.App{
		Name: "bun",
		Commands: []*cli.Command{
			{
				Name:  "db",
				Usage: "database commands",
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "create migration tables",
						Action: func(c *cli.Context) error {
							return migrator.Init(c.Context, db)
						},
					},
					{
						Name:  "migrate",
						Usage: "migrate database",
						Action: func(c *cli.Context) error {
							return migrator.Migrate(c.Context, db)
						},
					},
					{
						Name:  "rollback",
						Usage: "rollback the last migration batch",
						Action: func(c *cli.Context) error {
							return migrator.Rollback(c.Context, db)
						},
					},
					{
						Name:  "unlock",
						Usage: "unlock migrations",
						Action: func(c *cli.Context) error {
							return migrator.Unlock(c.Context, db)
						},
					},
					{
						Name:  "create_go",
						Usage: "create a Go migration",
						Action: func(c *cli.Context) error {
							return migrator.CreateGo(c.Context, db, c.Args().Get(0))
						},
					},
					{
						Name:  "create_sql",
						Usage: "create a SQL migration",
						Action: func(c *cli.Context) error {
							return migrator.CreateSQL(c.Context, db, c.Args().Get(0))
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
