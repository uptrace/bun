package buncli

import (
	"database/sql"
	"fmt"
	"log"
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
			flagNoCmd,
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
	defaultLoc           = "."
	defaultBin           = "bun"
	defaultMigrationsDir = "migrations"
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

	flagNoCmd = &cli.BoolFlag{
		Name:  "no-cmd",
		Usage: "don't create a CLI entrypoint in cmd/ directory",
		Value: false,
	}
)

func runInit(ctx *cli.Context, c *Config) error {
	m := migrate.NewMigrator(c.DB, c.Migrations)
	if err := m.Init(ctx.Context); err != nil {
		return err
	}

	loc := ctx.Args().Get(0)
	migrationsDir := loc

	if loc == "" {
		loc = defaultLoc
	}

	if loc == defaultLoc {
		migrationsDir = defaultMigrationsDir
	}

	log.Print("loc-0: " + loc)
	loc = path.Join(loc, "cmd", defaultBin)
	log.Print("loc: " + loc)

	if withCmd := !flagNoCmd.Get(ctx); withCmd {
		migrationsDir = path.Join(loc, migrationsDir)
		if err := initCmd(loc, migrationsDir); err != nil {
			return err
		}
	}
	log.Print("migrationsDir: " + migrationsDir)

	if err := initMigrationsPackage(migrationsDir); err != nil {
		return err
	}
	return nil
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

	if err := buncli.Run(os.Args, &buncli.Config{
		// DB: db,
		// AutoMigrator: auto,
		Migrations: migrations.Migrations,
	}); err != nil {
		panic(err)
	}
}
`

func initCmd(binDir string, migrationsDir string) error {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	log.Print("binDir: ", binDir)
	f, err := os.OpenFile(path.Join(binDir, "main.go"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		if os.IsExist(err) {
			// TODO: log the fact that we haven't modified an existing main.go
			return nil
		}
		return err
	}
	defer f.Close()

	modPath, err := getModPath()
	if err != nil {
		return err
	}
	log.Print("go.mod path: ", modPath)

	pkgMigrations := path.Join(modPath, strings.TrimLeft(migrationsDir, "."))
	log.Print("pkgMigrations: ", pkgMigrations)
	if _, err := fmt.Fprintf(f, entrypointTemplate, pkgMigrations); err != nil {
		log.Print("here!")
		return err
	}

	return nil
}

var migrationsTemplate = `package migrations

import "github.com/uptrace/bun/migrate"

var Migrations = migrate.NewMigrations()

func init() {
	if err := Migrations.DiscoverCaller(); err != nil {
		panic(err)
	}
}
`

func initMigrationsPackage(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path.Join(dir, "main.go"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		if os.IsExist(err) {
			// TODO: log the fact that we haven't modified an existing main.go
			return nil
		}
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprint(f, migrationsTemplate); err != nil {
		return err
	}
	return nil
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
