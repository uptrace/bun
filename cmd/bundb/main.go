/*
TODO:

- Add a mechanism to detect potentially duplicate migration files. That is,
once we've collected migrations in a bytes.Buffer, check if 'migrations/' package
has another migration files that:
 1. is identical in content
 2. belongs to the migration that has not been applied yet

If we find such migration, prompt the user for confirmation, unless -force flag is set.
Ideally, we should be able to ignore "transactional" for this purpose,
i.e. same_thing.up.tx.sql should overwrite same_thing.up.sql.

- Store configured options to env variables? E.g. after 'bundb init --create-directory=db-migrations/'
set BUNDB_MIGRATIONS=db-migrations, so that subsequent commands can be run without additional parameters.
Although... this way we are moving towards a .bundb.config or something.

- Allow defining components in the plugin, rather than passing config for them. Specifically:
 1. func DB() *bun.DB to return a database connection
    Handy in avoiding having to provide options for all the dialect-specific options here + potentially
    let's users re-use their existing "ConnectToDB" function.
 2. func AutoMigrator() *migrate.AutoMigrator to return a pre-configured AutoMigrator.
 3. ???
*/
package main

import (
	"bytes"
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
	autoMigratorOptions []migrate.AutoMigratorOption
	migrationsDirectory string
)

var (
	cleanup = &cli.BoolFlag{
		Name: "cleanup",
	}
)

var app = &cli.App{
	Name:  "bundb",
	Usage: "Database migration tool for uptrace/bun",
	Commands: cli.Commands{
		// bundb init --create-directory
		// bundb create --sql --go --tx [-d | --dir]
		// bundb migrate
		// bundb auto create --tx
		// bundb auto migrate
		&cli.Command{
			Name:  "auto",
			Usage: "manage database schema with AutoMigrator",
			Subcommands: cli.Commands{
				&cli.Command{
					Name:  "create",
					Usage: "Generate SQL migration files",
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
						&cli.BoolFlag{
							Name:    "tx",
							Aliases: []string{"transactional"},
						},
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

						if ctx.Bool("tx") {
							_, err = m.CreateTxSQLMigrations(ctx.Context)
						} else {
							_, err = m.CreateSQLMigrations(ctx.Context)
						}
						if err != nil {
							return err
						}
						return nil
					},
				},
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

// TODO: wrap Build and Open steps into a sync.OnceFunc, so that we could use the Plugin object in multiple places
// without having to worry if it has been compiled or not.
func buildPlugin(force bool) error {
	if force {
		if err := deletePlugin(); err != nil {
			return err
		}
	}

	// Cmd.Run returns *exec.ExitError which will only contain the exit code message in case of an error.
	// Rather than logging "exit code 1" we want to output a more informative error, so we redirect the Stderr.
	var errBuf bytes.Buffer

	cmd := exec.Command("go", "build", "-C", migrationsDirectory, "-buildmode", "plugin", "-o", pluginName)
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		// TODO: if errBuf contains "no such file or directory" add the following to the error message:
		// "Create 'migrations/' directory by running: bundb init --create-directory migrations/"
		return fmt.Errorf("build %s: %s", pluginPath(), &errBuf)
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
//	"postgres://root:@localhost:5432/test" -> "postgres"
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
