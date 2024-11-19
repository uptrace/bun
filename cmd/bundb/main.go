package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"plugin"
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
)

const (
	defaultMigrationsDirectory = "./migrations"
	pluginName                 = "plugin.so"
)

var (
	supportedDrivers    = []string{"postgres", "sqlserver", "mysql", "oci8", "file"}
	migrationsDirectory string

	// AutoMigrator options
	autoMigratorOptions []migrate.AutoMigratorOption
)

var (
	cleanup = &cli.BoolFlag{
		Name: "cleanup",
	}
)

var app = &cli.App{
	Name: "bundb",
	Commands: cli.Commands{
		// bundb init --create-directory
		// bundb create --sql --go --tx [-d | --dir]
		// bundb migrate
		// bundb auto create --sql --tx
		// bundb auto migrate
		&cli.Command{
			Name:  "auto",
			Usage: "manage database schema with AutoMigrator",
			Subcommands: cli.Commands{
				&cli.Command{
					Name:  "migrate",
					Usage: "Generate SQL migrations and apply them right away",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "uri",
							Aliases:  []string{"database-uri", "dsn"},
							Required: true,
							EnvVars:  []string{"BUNDB_URI"},
						},
						&cli.StringFlag{
							Name: "driver",
						},
						&cli.StringFlag{
							Name:        "d",
							Aliases:     []string{"migrations-directory"},
							Destination: &migrationsDirectory,
							Value:       defaultMigrationsDirectory,
							Action: func(ctx *cli.Context, dir string) error {
								autoMigratorOptions = append(autoMigratorOptions, migrate.WithMigrationsDirectoryAuto(dir))
								return nil
							},
						},
						&cli.StringFlag{
							Name:    "t",
							Aliases: []string{"migrations-table"},
							Action: func(ctx *cli.Context, migrationsTable string) error {
								autoMigratorOptions = append(autoMigratorOptions, migrate.WithTableNameAuto(migrationsTable))
								return nil
							},
						},
						&cli.StringFlag{
							Name:    "l",
							Aliases: []string{"locks", "migration-locks-table"},
							Action: func(ctx *cli.Context, locksTable string) error {
								autoMigratorOptions = append(autoMigratorOptions, migrate.WithLocksTableNameAuto(locksTable))
								return nil
							},
						},
						&cli.StringFlag{
							Name:    "s",
							Aliases: []string{"schema"},
							Action: func(ctx *cli.Context, schemaName string) error {
								autoMigratorOptions = append(autoMigratorOptions, migrate.WithSchemaName(schemaName))
								return nil
							},
						},
						&cli.StringSliceFlag{
							Name: "exclude",
							Action: func(ctx *cli.Context, tables []string) error {
								autoMigratorOptions = append(autoMigratorOptions, migrate.WithExcludeTable(tables...))
								return nil
							},
						},
						&cli.BoolFlag{
							Name: "rebuild",
						},
						cleanup,
					},
					Action: func(ctx *cli.Context) error {
						if err := buildPlugin(ctx.Bool("rebuild")); err != nil {
							return err
						}

						if cleanup.Get(ctx) {
							defer deletePlugin()
						}

						db, err := connect(ctx.String("uri"), ctx.String("driver"), !ctx.IsSet("driver"))
						if err != nil {
							return err

						}

						if !ctx.IsSet("migrations-directory") {
							autoMigratorOptions = append(autoMigratorOptions, migrate.WithMigrationsDirectoryAuto(defaultMigrationsDirectory))

						}
						m, err := automigrator(db)
						if err != nil {
							return err
						}

						group, err := m.Migrate(ctx.Context)
						if err != nil {
							return err
						}
						if group.IsZero() {
							log.Print("ok, nothing to migrate")
						}
						return nil
					},
				},
			},
		},
	},
}

func pluginPath() string {
	return path.Join(migrationsDirectory, pluginName)
}

func buildPlugin(force bool) error {
	if force {
		if err := deletePlugin(); err != nil {
			return err
		}
	}

	cmd := exec.Command("go", "build", "-C", migrationsDirectory, "-buildmode", "plugin", "-o", pluginName)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("build %s plugin: %w", pluginPath(), err)
	}
	return nil
}

func deletePlugin() error {
	return os.RemoveAll(pluginPath())
}

// connect to the database under the URI. A driver must be one of the supported drivers.
// If not set explicitly, the name of the driver is guessed from the URI.
//
// Example:
//
//	"postgres://postgres:@localhost:5432/postgres" -> "postegres"
func connect(uri, driverName string, guessDriver bool) (*bun.DB, error) {
	var sqldb *sql.DB
	var dialect schema.Dialect
	var err error

	if guessDriver {
		driver, _, found := strings.Cut(uri, ":")
		if !found {
			return nil, fmt.Errorf("driver cannot be guessed from connection string; pass -driver option explicitly")
		}
		driverName = driver
	}

	switch driverName {
	case "postgres":
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(uri)))
		dialect = pgdialect.New()
	case "sqlserver":
		sqldb, err = sql.Open(driverName, uri)
		dialect = mssqldialect.New()
	case "file":
		sqldb, err = sql.Open(sqliteshim.ShimName, uri)
		dialect = sqlitedialect.New()
	case "mysql":
		sqldb, err = sql.Open(driverName, uri)
		dialect = mysqldialect.New()
	case "oci8":
		sqldb, err = sql.Open(driverName, uri)
		dialect = oracledialect.New()
	default:
		err = fmt.Errorf("driver %q not recognized, supported drivers are %+v", driverName, supportedDrivers)
	}

	if err != nil {
		return nil, err
	}

	return bun.NewDB(sqldb, dialect), nil
}

// automigrator creates AutoMigrator for models from user's 'migrations' package.
func automigrator(db *bun.DB) (*migrate.AutoMigrator, error) {
	sym, err := lookup("Models")
	if err != nil {
		return nil, err
	}

	models, ok := sym.(*[]interface{})
	if !ok {
		return nil, fmt.Errorf("migrations plugin must export Models as []interface{}, got %T", models)
	}
	autoMigratorOptions = append(autoMigratorOptions, migrate.WithModel(*models...))

	auto, err := migrate.NewAutoMigrator(db, autoMigratorOptions...)
	if err != nil {
		return nil, err
	}
	return auto, nil
}

// lookup a symbol from user's migrations plugin.
func lookup(symbol string) (plugin.Symbol, error) {
	p, err := plugin.Open(pluginPath())
	if err != nil {
		return nil, err
	}
	return p.Lookup(symbol)
}

func main() {
	log.SetPrefix("bundb: ")
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
