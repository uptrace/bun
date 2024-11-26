package buncli

import (
	"github.com/urfave/cli/v2"
)

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
					return autoCreate(ctx, c)
				},
			},
			&cli.Command{
				Name:  "migrate",
				Usage: "Generate SQL migrations and apply them right away",
				Action: func(ctx *cli.Context) error {
					return autoMigrate(ctx, c)
				},
			},
		},
	}
}

var (
	// flagTx adds --transactional flag.
	flagTx = &cli.BoolFlag{
		Name:    "tx",
		Aliases: []string{"transactional"},
		Usage: "write migrations to .tx.(up|down).sql file, they will be marked as transactional",
		Value:   false,
	}
)

func autoMigrate(ctx *cli.Context, c *Config) error {
	_, err := c.AutoMigrator.Migrate(ctx.Context)
	return err
}

func autoCreate(ctx *cli.Context, c *Config) error {
	var err error
	if flagTx.Get(ctx) {
		_, err = c.AutoMigrator.CreateTxSQLMigrations(ctx.Context)
	} else {
		_, err = c.AutoMigrator.CreateSQLMigrations(ctx.Context)
	}
	return err
}
