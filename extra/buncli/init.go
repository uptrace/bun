package buncli

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mssqldialect"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/oracledialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/schema"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/modfile"
)

// CmdInit creates init command.
func CmdInit() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create migration tables and default project layout",
		Args:  true,
		Flags: []cli.Flag{
			flagDSN,
			flagDriver,
			flagBinary,
			flagMigrations,
			flagPluginMode,
		},
		Action: func(ctx *cli.Context) error {
			c, err := fromCLI(ctx)
			if err != nil {
				return err
			}
			return runInit(ctx, c)
		},
	}
}

const (
	maingo            = "main.go"
	defaultBin        = "bun"
	defaultMigrations = "migrations"
)

var (
	supportedDrivers = []string{"postgres", "sqlserver", "mysql", "oci8", "file"}

	flagDSN = &cli.StringFlag{
		Name:     "dsn",
		Usage:    "database connection string",
		Required: true,
		EnvVars:  []string{"BUNDB_URI"},
		Aliases:  []string{"uri"},
	}

	flagDriver = &cli.StringFlag{
		Name:  "driver",
		Usage: "database driver",
	}

	flagPluginMode = &cli.BoolFlag{
		Name:    "p",
		Aliases: []string{"plugin"},
		Usage:   "create a 'main' package to be used as a plugin",
	}

	flagBinary = &cli.StringFlag{
		Name:    "b",
		Aliases: []string{"binary"},
		Usage:   "name of the cmd/ binary",
		Value:   defaultBin,
	}

	flagMigrations = &cli.StringFlag{
		Name:    "m",
		Aliases: []string{"migrations-package"},
		Usage:   "name of the migrations package",
		Value:   defaultMigrations,
	}
)

func runInit(ctx *cli.Context, c *Config) error {
	m := migrate.NewMigrator(c.DB, c.Migrations)
	if err := m.Init(ctx.Context); err != nil {
		return err
	}

	loc := ctx.Args().Get(0)
	binName := flagBinary.Get(ctx)
	migrationsDir := flagMigrations.Get(ctx)

	var b interface{ Bootstrap() error }
	switch {
	default:
		b = &normalMode{Loc: loc, Binary: binName, Migrations: migrationsDir}
	case flagPluginMode.Get(ctx):
		b = &pluginMode{Loc: loc, Migrations: migrationsDir}
	}

	return b.Bootstrap()
}

// fromCLI creates minimal Config from command line arguments.
// It is inteded to be used exclusively by Init command, as it creates
// the default project structure and the user has no other way of configuring buncli.
//
// DB and Migrations are the only valid fields in the created config, other objects are nil.
func fromCLI(ctx *cli.Context) (*Config, error) {
	db, err := newDB(ctx)
	if err != nil {
		return nil, err
	}
	return &Config{DB: db, Migrations: migrate.NewMigrations()}, nil
}

type normalMode struct {
	Loc        string
	Binary     string
	Migrations string
}

const entrypointTemplate = `package main

import (
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/buncli"
	"github.com/uptrace/bun/migrate"

	%q
)

func main() {
	// TODO: connect to db
	var _ /* db */ *bun.DB
	
	// TODO: configure AutoMigrator
	var _ /* auto */ migrate.AutoMigrator

	cfg := buncli.Config{
		RootName: %q,
		// DB: db,
		// AutoMigrator: auto,
		Migrations: migrations.Migrations,
	}

	if err := buncli.Run(os.Args, &cfg); err != nil {
		panic(err)
	}
}
`

var migrationsTemplate = `package migrations

import "github.com/uptrace/bun/migrate"

var Migrations = migrate.NewMigrations()

func init() {
	if err := Migrations.DiscoverCaller(); err != nil {
		panic(err)
	}
}
`

func (n *normalMode) Bootstrap() error {
	// Create cmd/bun/main.go entrypoint
	binDir := path.Join(n.Loc, "cmd", n.Binary)
	modPath, err := n.pkgMigrations(binDir)
	if err != nil {
		return err
	}
	if err := writef(binDir, maingo, entrypointTemplate, modPath, n.Binary); err != nil {
		return err
	}

	// Create migrations/main.go template
	migrationsDir := path.Join(binDir, n.Migrations)
	if err := writef(migrationsDir, maingo, migrationsTemplate); err != nil {
		return err
	}
	return nil
}

func (n *normalMode) pkgMigrations(binDir string) (string, error) {
	modPath, err := getModPath()
	if err != nil {
		return "", err
	}
	return path.Join(modPath, strings.TrimLeft(binDir, "."), n.Migrations), nil
}

type pluginMode struct {
	Loc        string
	Migrations string
}

var pluginTemplate = `package main

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/buncli"
	"github.com/uptrace/bun/migrate"
)

var Config *buncli.Config

func init() {
	migrations := migrate.NewMigrations()
	if err := migrations.DiscoverCaller(); err != nil {
		panic(err)
	}

	// TODO: connect to db
	var _ /* db */ *bun.DB
	
	// TODO: configure AutoMigrator
	var _ /* auto */ migrate.AutoMigrator

	Config = &buncli.Config{
		// DB: db,
		// AutoMigrator: auto,
		Migrations: migrations,
	}
}
`

func (p *pluginMode) Bootstrap() error {
	binDir := path.Join(p.Loc, p.Migrations)
	if err := writef(binDir, maingo, pluginTemplate); err != nil {
		return err
	}
	return nil
}

// getModPath parses the ./go.mod file in the current directory and returns the declared module path.
func getModPath() (string, error) {
	f, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	gomod, err := modfile.Parse("go.mod", f, nil)
	if err != nil {
		return "", err
	}
	return gomod.Module.Mod.Path, nil
}

// TODO: document
func writef(dir string, file string, format string, args ...interface{}) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path.Join(dir, file), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		if os.IsExist(err) {
			// TODO: log the fact that we haven't modified an existing main.go
			return nil
		}
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, format, args...); err != nil {
		return err
	}
	return nil
}

// newDB connects to the database specified by the DSN.
// It will attempt to guess the driver from the connection string prefix, unless it is passed explicitly.
func newDB(ctx *cli.Context) (*bun.DB, error) {
	var sqlDB *sql.DB
	var dialect schema.Dialect
	var err error

	dsn := flagDSN.Get(ctx)
	driver := flagDriver.Get(ctx)
	if !flagDriver.IsSet() {
		guess, _, found := strings.Cut(dsn, ":")
		if !found {
			return nil, fmt.Errorf("driver cannot be guessed from connection string; pass --driver option explicitly")
		}
		driver = guess
	}

	switch driver {
	case "postgres":
		sqlDB = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		dialect = pgdialect.New()
	case "sqlserver":
		sqlDB, err = sql.Open(driver, dsn)
		dialect = mssqldialect.New()
	case "file":
		sqlDB, err = sql.Open(sqliteshim.ShimName, dsn)
		dialect = sqlitedialect.New()
	case "mysql":
		sqlDB, err = sql.Open(driver, dsn)
		dialect = mysqldialect.New()
	case "oci8":
		sqlDB, err = sql.Open(driver, dsn)
		dialect = oracledialect.New()
	default:
		err = fmt.Errorf("driver %q not recognized, supported drivers are %v", driver, supportedDrivers)
	}

	if err != nil {
		return nil, err
	}

	return bun.NewDB(sqlDB, dialect), nil
}
