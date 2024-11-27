package buncli

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"plugin"
	"sync"

	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

const (
	pluginName       = "plugin.so"
	configLookupName = "Config"
)

var (
	buildOnce  = sync.OnceValue(buildPlugin)
	importOnce = sync.OnceValues(importConfig)

	pluginPath string

	flagPluginPath = &cli.StringFlag{
		Name:        "m",
		Usage:       "relative `PATH` to migrations directory",
		Value:       "./" + defaultMigrations,
		Destination: &pluginPath,
	}
)

func pluginCommands() cli.Commands {
	c := fromPlugin()

	auto := CmdAuto(c)
	skipAuto := func(c *cli.Command) bool {
		return c == auto
	}
	return extendCommands(cli.Commands{
		auto,
		CmdMigrate(c),
		CmdRollback(c),
		CmdCreate(c),
		CmdUnlock(c),
	}, skipAuto, withBefore(checkCanImportPlugin), withFlag(flagPluginPath))
}

func fromPlugin() *pluginConfig {
	return &pluginConfig{
		config: func() *Config {
			c, _ := importOnce()
			return c
		},
	}
}

// checkCanImportPlugin returns an error if migrations/ plugin build failed or *buncli.Config could not be imported from it.
func checkCanImportPlugin(ctx *cli.Context) error {
	if isHelp(ctx) {
		// Do not build plugin if on -help.
		return nil
	}
	_, err := importOnce()
	return err
}

func isHelp(ctx *cli.Context) bool {
	help := cli.HelpFlag.(*cli.BoolFlag)
	return ctx.Command.Name == bunApp.HelpName || help.Get(ctx)
}

// importConfig builds migrations/ plugin and imports Config from it.
// Config must be exported and of type *buncli.Config.
func importConfig() (*Config, error) {
	if err := buildOnce(); err != nil {
		return nil, err
	}

	p, err := plugin.Open(path.Join(pluginPath, pluginName))
	if err != nil {
		return nil, err
	}
	sym, err := p.Lookup(configLookupName)
	if err != nil {
		return nil, err
	}
	cfg, ok := sym.(**Config)
	if !ok {
		return nil, fmt.Errorf("migrations plugin must export Config as *buncli.Config, got %T", sym)
	}
	return *cfg, nil
}

type pluginConfig struct {
	config func() *Config
}

var _ OptionsConfig = (*pluginConfig)(nil)
var _ MigratorConfig = (*pluginConfig)(nil)
var _ AutoConfig = (*pluginConfig)(nil)

func (p *pluginConfig) NewMigrator() *migrate.Migrator {
	return p.config().NewMigrator()
}

func (p *pluginConfig) Auto() *migrate.AutoMigrator {
	return p.config().Auto()
}

func (p *pluginConfig) GetMigratorOptions() []migrate.MigratorOption {
	return p.config().GetMigratorOptions()
}

func (p *pluginConfig) GetMigrateOptions() []migrate.MigrationOption {
	return p.config().GetMigrateOptions()
}

func (p *pluginConfig) GetGoMigrationOptions() []migrate.GoMigrationOption {
	return p.config().GetGoMigrationOptions()
}

// buildPlugin compiles migrations/ plugin with -buildmode=plugin.
func buildPlugin() error {
	cmd := exec.Command("go", "build", "-C", pluginPath, "-buildmode", "plugin", "-o", pluginName)

	// Cmd.Run returns *exec.ExitError which will only contain the exit code message in case of an error.
	// Rather than logging "exit code 1" we want to output a more informative error, so we redirect the Stderr.
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("build %s: %s", path.Join(pluginPath, pluginName), &errBuf)
	}
	return nil
}

type commandOption func(*cli.Command)

func extendCommands(commands cli.Commands, skip func(c *cli.Command) bool, options ...commandOption) cli.Commands {
	var all cli.Commands
	var flatten func(cmds cli.Commands)

	flatten = func(cmds cli.Commands) {
		for _, cmd := range cmds {
			if skip != nil && !skip(cmd) {
				all = append(all, cmd)
			}
			flatten(cmd.Subcommands)
		}
	}
	flatten(commands)

	for _, cmd := range all {
		for _, opt := range options {
			opt(cmd)
		}
	}
	return commands
}

// withFlag adds a flag to command.
func withFlag(f cli.Flag) commandOption {
	return func(c *cli.Command) {
		c.Flags = append(c.Flags, f)
	}
}

// withBefore adds BeforeFunc to command.
func withBefore(bf cli.BeforeFunc) commandOption {
	return func(c *cli.Command) {
		c.Before = bf
	}
}
