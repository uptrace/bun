package buncli

import (
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

// CmdAuto creates the auto command hierarchy.
func CmdAuto(c AutoConfig) *cli.Command {
	return &cli.Command{
		Name:  "auto",
		Usage: "Manage database schema with AutoMigrator",
		Subcommands: cli.Commands{
			&cli.Command{
				Name:  "create",
				Usage: "Generate SQL migration files",
				Flags: []cli.Flag{
					flagTx,
				},
				Action: func(ctx *cli.Context) error {
					return runAutoCreate(ctx, c)
				},
			},
			&cli.Command{
				Name:  "migrate",
				Usage: "Generate SQL migrations and apply them right away",
				Action: func(ctx *cli.Context) error {
					return runAutoMigrate(ctx, c)
				},
			},
		},
	}
}

// AutoConfig provides configuration for commands related to auto migration.
type AutoConfig interface {
	OptionsConfig
	Auto() *migrate.AutoMigrator
}

func runAutoMigrate(ctx *cli.Context, c AutoConfig) error {
	_, err := c.Auto().Migrate(ctx.Context, c.GetMigrateOptions()...)
	return err
}

func runAutoCreate(ctx *cli.Context, c AutoConfig) error {
	var err error
	if flagTx.Get(ctx) {
		_, err = c.Auto().CreateTxSQLMigrations(ctx.Context)
	} else {
		_, err = c.Auto().CreateSQLMigrations(ctx.Context)
	}
	return err
}
