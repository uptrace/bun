package buncli

import (
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

// CmdMigrate creates migrate command.
func CmdMigrate(c *Config) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Apply database migrations",
		Action: func(ctx *cli.Context) error {
			return runMigrate(ctx, c)
		},
	}
}

func runMigrate(ctx *cli.Context, c *Config) error {
	m := migrate.NewMigrator(c.DB, c.Migrations, c.MigratorOptions...)
	_, err := m.Migrate(ctx.Context, c.MigrateOptions...)
	return err
}

// CmdRollback creates rollback command.
func CmdRollback(c *Config) *cli.Command {
	return &cli.Command{
		Name:  "rollback",
		Usage: "Rollback the last migration group",
		Action: func(ctx *cli.Context) error {
			return runRollback(ctx, c)
		},
	}
}

func runRollback(ctx *cli.Context, c *Config) error {
	m := migrate.NewMigrator(c.DB, c.Migrations, c.MigratorOptions...)
	_, err := m.Rollback(ctx.Context, c.MigrateOptions...)
	return err
}

// CmdCreate creates create command.
func CmdCreate(c *Config) *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new migration file template",
		Args:  true, // TODO: add usage example and description
		Flags: []cli.Flag{
			flagTx,
			flagGo,
			flagSQL,
		},
		Action: func(ctx *cli.Context) error {
			return runCreate(ctx, c)
		},
	}
}

var (
	// createGo controls the type of the template (Go/SQL) the Migrator should create.
	createGo bool

	// flagTx adds --transactional flag.
	flagTx = &cli.BoolFlag{
		Name:    "tx",
		Aliases: []string{"transactional"},
		Usage:   "write migrations to .tx.(up|down).sql file, they will be marked as transactional",
		Value:   false,
	}

	// flagGo adds --go flag. Prefer checking 'createGo' as the flagGo and flagSQL are mutually exclusive and it is how they synchronize.
	flagGo = &cli.BoolFlag{
		Name:        "go",
		Usage:       "create .go migration file",
		Value:       false,
		Destination: &createGo,
	}

	// flagSQL adds --sql flag. Prefer checking 'createGo' as the flagGo and flagSQL are mutually exclusive and it is how they synchronize.
	flagSQL = &cli.BoolFlag{
		Name:  "sql",
		Usage: "create .sql migrations",
		Value: true,
		Action: func(ctx *cli.Context, b bool) error {
			createGo = !b
			return nil
		},
	}
)

func runCreate(ctx *cli.Context, c *Config) error {
	var err error
	m := migrate.NewMigrator(c.DB, c.Migrations, c.MigratorOptions...)
	name := ctx.Args().First()

	if createGo {
		_, err = m.CreateGoMigration(ctx.Context, name, c.GoMigrationOptions...)
		return err
	}

	if flagTx.Get(ctx) {
		_, err = m.CreateTxSQLMigrations(ctx.Context, name)
	} else {
		_, err = m.CreateSQLMigrations(ctx.Context, name)
	}
	return err
}

// CmdUnlock creates an unlock command.
func CmdUnlock(c *Config) *cli.Command {
	return &cli.Command{
		Name:  "unlock",
		Usage: "Unlock migration locks table",
		Action: func(ctx *cli.Context) error {
			return runUnlock(ctx, c)
		},
	}
}

func runUnlock(ctx *cli.Context, c *Config) error {
	m := migrate.NewMigrator(c.DB, c.Migrations, c.MigratorOptions...)
	return m.Unlock(ctx.Context)
}
