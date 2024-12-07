/*
TODO:
  - Commands:
  - init - Create migration+locks tables [--no-cmd to omit cmd/ folder]
  - provide NewCommand() *cli.Command intead of the cli.App, so that buncli could be embeded in the existing CLIs
  - configure logging and verbosity
  - (experimental, low prio) add FromPlugin() to read config from plugin and use from cmd/bundb.
*/
package buncli

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

// bunApp is the root-level bundb app that all other commands attach to.
var bunApp = &cli.App{
	Name:    "bundb",
	Usage:   "Database migration tool for uptrace/bun",
	Suggest: true,
}

// New creates a new CLI application for managing bun migrations.
func New(c *Config) *App {
	if c.RootName != "" {
		bunApp.Name = c.RootName
	}
	bunApp.Commands = cli.Commands{
		CmdMigrate(c),
		CmdRollback(c),
		CmdCreate(c),
		CmdAuto(c),
		CmdUnlock(c),
	}
	return &App{
		App: bunApp,
	}
}

// NewStandalone create a new CLI application to be distributed as a standalone binary.
// It's intended to be used in the cmb/bundb and does not require any prior setup from the user:
// the app only includes the Init command and reads all its configuration from command line.
//
// Prefer using New(*Config) in your custom entrypoint.
func NewStandalone(name string) *App {
	bunApp.Name = name
	bunApp.Commands = cli.Commands{
		CmdInit(),
	}

	// NOTE: use `-tags experimental` to enable/disable this feature?
	addCommandGroup(bunApp, "EXPERIMENTAL", pluginCommands()...)
	return &App{
		App: bunApp,
	}
}

type Config struct {
	RootName           string
	DB                 *bun.DB
	AutoMigrator       *migrate.AutoMigrator
	Migrations         *migrate.Migrations
	MigratorOptions    []migrate.MigratorOption
	MigrateOptions     []migrate.MigrationOption
	GoMigrationOptions []migrate.GoMigrationOption
}

// Run calls cli.App.Run and returns its error.
func Run(args []string, c *Config) error {
	return New(c).Run(args)
}

// RunContext calls cli.App.RunContext and returns its error.
func RunContext(ctx context.Context, args []string, c *Config) error {
	return New(c).RunContext(ctx, args)
}

// App is a wrapper around cli.App that extends it with bun-specific features.
type App struct {
	*cli.App
}

var _ OptionsConfig = (*Config)(nil)
var _ MigratorConfig = (*Config)(nil)
var _ AutoConfig = (*Config)(nil)

func (c *Config) NewMigrator() *migrate.Migrator {
	return migrate.NewMigrator(c.DB, c.Migrations, c.MigratorOptions...)
}

func (c *Config) Auto() *migrate.AutoMigrator                        { return c.AutoMigrator }
func (c *Config) GetMigratorOptions() []migrate.MigratorOption       { return c.MigratorOptions }
func (c *Config) GetMigrateOptions() []migrate.MigrationOption       { return c.MigrateOptions }
func (c *Config) GetGoMigrationOptions() []migrate.GoMigrationOption { return c.GoMigrationOptions }

// addCommandGroup groups commands into one category.
func addCommandGroup(app *cli.App, group string, commands ...*cli.Command) {
	for _, cmd := range commands {
		cmd.Category = group
	}
	app.Commands = append(app.Commands, commands...)
}