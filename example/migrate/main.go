package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/example/migrate/migrations"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"

	"github.com/urfave/cli/v2"

	"github.com/uptrace/bun"
)

func main() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file:test.s3db?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

	app := &cli.App{
		Name: "bun",

		Commands: []*cli.Command{
			newDBCommand(migrate.NewMigrator(db, migrations.Migrations)),
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newDBCommand(migrator *migrate.Migrator) *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "create migration tables",
				Action: func(c *cli.Context) error {
					return migrator.Init(c.Context)
				},
			},
			{
				Name:  "migrate",
				Usage: "migrate database",
				Action: func(c *cli.Context) error {
					group, err := migrator.Migrate(c.Context)
					if err != nil {
						return err
					}
					if group.ID == 0 {
						fmt.Printf("there are no new migrations to run\n")
						return nil
					}
					fmt.Printf("migrated to %s\n", group)
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "rollback the last migration group",
				Action: func(c *cli.Context) error {
					group, err := migrator.Rollback(c.Context)
					if err != nil {
						return err
					}
					if group.ID == 0 {
						fmt.Printf("there are no groups to roll back\n")
						return nil
					}
					fmt.Printf("rolled back %s\n", group)
					return nil
				},
			},
			{
				Name:  "lock",
				Usage: "lock migrations",
				Action: func(c *cli.Context) error {
					return migrator.Lock(c.Context)
				},
			},
			{
				Name:  "unlock",
				Usage: "unlock migrations",
				Action: func(c *cli.Context) error {
					return migrator.Unlock(c.Context)
				},
			},
			{
				Name:  "create_go",
				Usage: "create Go migration",
				Action: func(c *cli.Context) error {
					mf, err := migrator.CreateGo(c.Context, c.Args().Get(0))
					if err != nil {
						return err
					}
					fmt.Printf("created migration %s (%s)\n", mf.FileName, mf.FilePath)
					return nil
				},
			},
			{
				Name:  "create_sql",
				Usage: "create SQL migration",
				Action: func(c *cli.Context) error {
					mf, err := migrator.CreateSQL(c.Context, c.Args().Get(0))
					if err != nil {
						return err
					}
					fmt.Printf("created migration %s (%s)\n", mf.FileName, mf.FilePath)
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "print migrations status",
				Action: func(c *cli.Context) error {
					status, err := migrator.Status(c.Context)
					if err != nil {
						return err
					}
					fmt.Printf("migrations: %s\n", status.Migrations)
					fmt.Printf("new migrations: %s\n", status.NewMigrations)
					fmt.Printf("last group: %s\n", status.LastGroup)
					return nil
				},
			},
			{
				Name:  "mark_completed",
				Usage: "mark migrations as completed without actually running them",
				Action: func(c *cli.Context) error {
					group, err := migrator.MarkCompleted(c.Context)
					if err != nil {
						return err
					}
					if group.ID == 0 {
						fmt.Printf("there are no new migrations to mark as completed\n")
						return nil
					}
					fmt.Printf("marked as completed %s\n", group)
					return nil
				},
			},
		},
	}
}
