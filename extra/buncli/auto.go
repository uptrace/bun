package buncli

import (
	"github.com/urfave/cli/v2"
)

// CmdAuto creates the auto command hierarchy.
func CmdAuto(c *Config) *cli.Command {
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

func runAutoMigrate(ctx *cli.Context, c *Config) error {
	_, err := c.AutoMigrator.Migrate(ctx.Context, c.MigrateOptions...)
	return err
}

func runAutoCreate(ctx *cli.Context, c *Config) error {
	var err error
	if flagTx.Get(ctx) {
		_, err = c.AutoMigrator.CreateTxSQLMigrations(ctx.Context)
	} else {
		_, err = c.AutoMigrator.CreateSQLMigrations(ctx.Context)
	}
	return err
}
