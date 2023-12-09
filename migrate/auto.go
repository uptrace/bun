package migrate

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

type AutoMigratorOption func(m *AutoMigrator)

// WithModel adds a bun.Model to the scope of migrations.
func WithModel(models ...interface{}) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.includeModels = append(m.includeModels, models...)
	}
}

// WithExcludeTable tells the AutoMigrator to ignore a table in the database.
func WithExcludeTable(tables ...string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.excludeTables = append(m.excludeTables, tables...)
	}
}

// WithFKNameFunc sets the function to build a new name for created or renamed FK constraints.
//
// Notice: this option is not supported in SQLite dialect and will have no effect.
// SQLite does not implement ADD CONSTRAINT, so adding or renaming a constraint will require re-creating the table.
// We need to support custom FKNameFunc in CreateTable to control how FKs are named.
//
// More generally, this option will have no effect whenever FKs are included in the CREATE TABLE definition,
// which is the default strategy. Perhaps it would make sense to allow disabling this and switching to separate (CreateTable + AddFK)
func WithFKNameFunc(f func(sqlschema.FK) string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.diffOpts = append(m.diffOpts, fKNameFunc(f))
	}
}

// WithRenameFK prevents AutoMigrator from recreating foreign keys when their dependent relations are renamed,
// and forces it to run a RENAME CONSTRAINT query instead. Creating an index on a large table can take a very long time,
// and in those cases simply renaming the FK makes a lot more sense.
func WithRenameFK(enabled bool) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.diffOpts = append(m.diffOpts, detectRenamedFKs(enabled))
	}
}

// WithTableNameAuto overrides default migrations table name.
func WithTableNameAuto(table string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.table = table
		m.migratorOpts = append(m.migratorOpts, WithTableName(table))
	}
}

// WithLocksTableNameAuto overrides default migration locks table name.
func WithLocksTableNameAuto(table string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.locksTable = table
		m.migratorOpts = append(m.migratorOpts, WithLocksTableName(table))
	}
}

// WithMarkAppliedOnSuccessAuto sets the migrator to only mark migrations as applied/unapplied
// when their up/down is successful.
func WithMarkAppliedOnSuccessAuto(enabled bool) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.migratorOpts = append(m.migratorOpts, WithMarkAppliedOnSuccess(enabled))
	}
}

type AutoMigrator struct {
	db *bun.DB

	// dbInspector creates the current state for the target database.
	dbInspector sqlschema.Inspector

	// modelInspector creates the desired state based on the model definitions.
	modelInspector sqlschema.Inspector

	// dbMigrator executes ALTER TABLE queries.
	dbMigrator sqlschema.Migrator

	table      string // Migrations table (excluded from database inspection)
	locksTable string // Migration locks table (excluded from database inspection)

	// includeModels define the migration scope.
	includeModels []interface{}

	// excludeTables are excluded from database inspection.
	excludeTables []string

	// diffOpts are passed to detector constructor.
	diffOpts []diffOption

	// migratorOpts are passed to Migrator constructor.
	migratorOpts []MigratorOption
}

func NewAutoMigrator(db *bun.DB, opts ...AutoMigratorOption) (*AutoMigrator, error) {
	am := &AutoMigrator{
		db:         db,
		table:      defaultTable,
		locksTable: defaultLocksTable,
	}

	for _, opt := range opts {
		opt(am)
	}
	am.excludeTables = append(am.excludeTables, am.table, am.locksTable)

	dbInspector, err := sqlschema.NewInspector(db, am.excludeTables...)
	if err != nil {
		return nil, err
	}
	am.dbInspector = dbInspector

	dbMigrator, err := sqlschema.NewMigrator(db)
	if err != nil {
		return nil, err
	}
	am.dbMigrator = dbMigrator

	tables := schema.NewTables(db.Dialect())
	tables.Register(am.includeModels...)
	am.modelInspector = sqlschema.NewSchemaInspector(tables)

	return am, nil
}

func (am *AutoMigrator) plan(ctx context.Context) (*changeset, error) {
	var err error

	got, err := am.dbInspector.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	want, err := am.modelInspector.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	detector := newDetector(got, want, am.diffOpts...)
	changes := detector.Diff()
	if err := changes.ResolveDependencies(); err != nil {
		return nil, fmt.Errorf("plan migrations: %w", err)
	}
	return changes, nil
}

// Migrate writes required changes to a new migration file and runs the migration.
// This will create and entry in the migrations table, making it possible to revert
// the changes with Migrator.Rollback().
func (am *AutoMigrator) Migrate(ctx context.Context, opts ...MigrationOption) error {
	changes, err := am.plan(ctx)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	migrations := NewMigrations()
	name, _ := genMigrationName("auto")
	migrations.Add(Migration{
		Name:    name,
		Up:      changes.Up(am.dbMigrator),
		Down:    changes.Down(am.dbMigrator),
		Comment: "Changes detected by bun.migrate.AutoMigrator",
	})

	migrator := NewMigrator(am.db, migrations, am.migratorOpts...)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	if _, err := migrator.Migrate(ctx, opts...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// Run runs required migrations in-place and without creating a database entry.
func (am *AutoMigrator) Run(ctx context.Context) error {
	changes, err := am.plan(ctx)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	up := changes.Up(am.dbMigrator)
	if err := up(ctx, am.db); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}