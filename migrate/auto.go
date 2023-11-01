package migrate

import (
	"context"
	"fmt"
	"strings"

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

	table      string
	locksTable string

	// includeModels define the migration scope.
	includeModels []interface{}

	// excludeTables are excluded from database inspection.
	excludeTables []string

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

func (am *AutoMigrator) diff(ctx context.Context) (Changeset, error) {
	var detector Detector
	var changes Changeset
	var err error

	got, err := am.dbInspector.Inspect(ctx)
	if err != nil {
		return changes, err
	}

	want, err := am.modelInspector.Inspect(ctx)
	if err != nil {
		return changes, err
	}
	return detector.Diff(got, want), nil
}

// Migrate writes required changes to a new migration file and runs the migration.
// This will create and entry in the migrations table, making it possible to revert
// the changes with Migrator.Rollback().
func (am *AutoMigrator) Migrate(ctx context.Context, opts ...MigrationOption) error {
	changeset, err := am.diff(ctx)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	migrations := NewMigrations()
	name, _ := genMigrationName("auto")
	migrations.Add(Migration{
		Name:    name,
		Up:      changeset.Up(am.dbMigrator),
		Down:    changeset.Down(am.dbMigrator),
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
	changeset, err := am.diff(ctx)
	if err != nil {
		return fmt.Errorf("run auto migrate: %w", err)
	}
	up := changeset.Up(am.dbMigrator)
	if err := up(ctx, am.db); err != nil {
		return fmt.Errorf("run auto migrate: %w", err)
	}
	return nil
}

// INTERNAL -------------------------------------------------------------------

// Operation is an abstraction a level above a MigrationFunc.
// Apart from storing the function to execute the change,
// it knows how to *write* the corresponding code, and what the reverse operation is.
type Operation interface {
	Func(sqlschema.Migrator) MigrationFunc
	// GetReverse returns an operation that can revert the current one.
	GetReverse() Operation
}

type RenameTable struct {
	From string
	To   string
}

func (rt *RenameTable) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.RenameTable(ctx, rt.From, rt.To)
	}
}

func (rt *RenameTable) GetReverse() Operation {
	return &RenameTable{
		From: rt.To,
		To:   rt.From,
	}
}

// Changeset is a set of changes that alter database state.
type Changeset struct {
	operations []Operation
}

var _ Operation = (*Changeset)(nil)

func (c Changeset) Operations() []Operation {
	return c.operations
}

func (c *Changeset) Add(op Operation) {
	c.operations = append(c.operations, op)
}

func (c *Changeset) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		for _, op := range c.operations {
			fn := op.Func(m)
			if err := fn(ctx, db); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *Changeset) GetReverse() Operation {
	var reverse Changeset
	for _, op := range c.operations {
		reverse.Add(op.GetReverse())
	}
	return &reverse
}

// Up is syntactic sugar.
func (c *Changeset) Up(m sqlschema.Migrator) MigrationFunc {
	return c.Func(m)
}

// Down is syntactic sugar.
func (c *Changeset) Down(m sqlschema.Migrator) MigrationFunc {
	return c.GetReverse().Func(m)
}

type Detector struct{}

func (d *Detector) Diff(got, want sqlschema.State) Changeset {
	var changes Changeset

	// Detect renamed models
	oldModels := newTableSet(got.Tables...)
	newModels := newTableSet(want.Tables...)

	addedModels := newModels.Sub(oldModels)
	for _, added := range addedModels.Values() {
		removedModels := oldModels.Sub(newModels)
		for _, removed := range removedModels.Values() {
			if !sqlschema.EqualSignatures(added, removed) {
				continue
			}
			changes.Add(&RenameTable{
				From: removed.Name,
				To:   added.Name,
			})
		}
	}

	return changes
}

// tableSet stores unique table definitions.
type tableSet struct {
	underlying map[string]sqlschema.Table
}

func newTableSet(initial ...sqlschema.Table) tableSet {
	set := tableSet{
		underlying: make(map[string]sqlschema.Table),
	}
	for _, t := range initial {
		set.Add(t)
	}
	return set
}

func (set tableSet) Add(t sqlschema.Table) {
	set.underlying[t.Name] = t
}

func (set tableSet) Remove(s string) {
	delete(set.underlying, s)
}

func (set tableSet) Values() (tables []sqlschema.Table) {
	for _, t := range set.underlying {
		tables = append(tables, t)
	}
	return
}

func (set tableSet) Sub(other tableSet) tableSet {
	res := set.clone()
	for v := range other.underlying {
		if _, ok := set.underlying[v]; ok {
			res.Remove(v)
		}
	}
	return res
}

func (set tableSet) clone() tableSet {
	res := newTableSet()
	for _, t := range set.underlying {
		res.Add(t)
	}
	return res
}

func (set tableSet) String() string {
	var s strings.Builder
	for k := range set.underlying {
		if s.Len() > 0 {
			s.WriteString(", ")
		}
		s.WriteString(k)
	}
	return s.String()
}
