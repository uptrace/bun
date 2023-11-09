package migrate

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	return Diff(got, want), nil
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

func Diff(got, want sqlschema.State) Changeset {
	detector := newDetector()
	return detector.DetectChanges(got, want)
}

type detector struct {
	changes Changeset
}

func newDetector() *detector {
	return &detector{}
}

func (d *detector) DetectChanges(got, want sqlschema.State) Changeset {

	// TableSets for discovering CREATE/RENAME/DROP TABLE
	oldModels := newTableSet(got.Tables...) //
	newModels := newTableSet(want.Tables...)

	addedModels := newModels.Sub(oldModels)

AddedLoop:
	for _, added := range addedModels.Values() {
		removedModels := oldModels.Sub(newModels)
		for _, removed := range removedModels.Values() {
			if d.canRename(added, removed) {
				d.changes.Add(&RenameTable{
					Schema: removed.Schema,
					From:   removed.Name,
					To:     added.Name,
				})

				// TODO: check for altered columns.

				// Do not check this model further, we know it was renamed.
				oldModels.Remove(removed.Name)
				continue AddedLoop
			}
		}
		// If a new table did not appear because of the rename operation, then it must've been created.
		d.changes.Add(&CreateTable{
			Schema: added.Schema,
			Name:   added.Name,
			Model:  added.Model,
		})
	}

	// Tables that aren't present anymore and weren't renamed were deleted.
	for _, t := range oldModels.Sub(newModels).Values() {
		d.changes.Add(&DropTable{
			Schema: t.Schema,
			Name:   t.Name,
		})
	}

	// Compare FKs
	for fk /*, fkName */ := range want.FKs {
		if _, ok := got.FKs[fk]; !ok {
			d.changes.Add(&AddForeignKey{
				SourceSchema:  fk.From.Schema,
				SourceTable:   fk.From.Table,
				SourceColumns: fk.From.Column.Split(),
				TargetSchema:  fk.To.Schema,
				TargetTable:   fk.To.Table,
				TargetColumns: fk.To.Column.Split(),
			})
		}
	}

	for fk, fkName := range got.FKs {
		if _, ok := want.FKs[fk]; !ok {
			d.changes.Add(&DropForeignKey{
				Schema:         fk.From.Schema,
				Table:          fk.From.Table,
				ConstraintName: fkName,
			})
		}
	}

	return d.changes
}

// canRename checks if t1 can be renamed to t2.
func (d detector) canRename(t1, t2 sqlschema.Table) bool {
	return t1.Schema == t2.Schema && sqlschema.EqualSignatures(t1, t2)
}

// Changeset is a set of changes that alter database state.
type Changeset struct {
	operations []Operation
}

var _ Operation = (*Changeset)(nil)

func (c Changeset) String() string {
	var ops []string
	for _, op := range c.operations {
		ops = append(ops, op.String())
	}
	if len(ops) == 0 {
		return ""
	}
	return strings.Join(ops, "\n")
}

func (c Changeset) Operations() []Operation {
	return c.operations
}

// Add new operations to the changeset.
func (c *Changeset) Add(op ...Operation) {
	c.operations = append(c.operations, op...)
}

// Func chains all underlying operations in a single MigrationFunc.
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

// Operation is an abstraction a level above a MigrationFunc.
// Apart from storing the function to execute the change,
// it knows how to *write* the corresponding code, and what the reverse operation is.
type Operation interface {
	fmt.Stringer

	Func(sqlschema.Migrator) MigrationFunc
	// GetReverse returns an operation that can revert the current one.
	GetReverse() Operation
}

// noop is a migration that doesn't change the schema.
type noop struct{}

var _ Operation = (*noop)(nil)

func (*noop) String() string { return "noop" }
func (*noop) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error { return nil }
}
func (*noop) GetReverse() Operation { return &noop{} }

type RenameTable struct {
	Schema string
	From   string
	To     string
}

var _ Operation = (*RenameTable)(nil)

func (op RenameTable) String() string {
	return fmt.Sprintf(
		"Rename table %q.%q to %q.%q",
		op.Schema, trimSchema(op.From), op.Schema, trimSchema(op.To),
	)
}

func (op *RenameTable) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.RenameTable(ctx, op.From, op.To)
	}
}

func (op *RenameTable) GetReverse() Operation {
	return &RenameTable{
		From: op.To,
		To:   op.From,
	}
}

type CreateTable struct {
	Schema string
	Name   string
	Model  interface{}
}

var _ Operation = (*CreateTable)(nil)

func (op CreateTable) String() string {
	return fmt.Sprintf("CreateTable %T", op.Model)
}

func (op *CreateTable) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.CreateTable(ctx, op.Model)
	}
}

func (op *CreateTable) GetReverse() Operation {
	return &DropTable{
		Schema: op.Schema,
		Name:   op.Name,
	}
}

type DropTable struct {
	Schema string
	Name   string
}

var _ Operation = (*DropTable)(nil)

func (op DropTable) String() string {
	return fmt.Sprintf("DropTable %q.%q", op.Schema, trimSchema(op.Name))
}

func (op *DropTable) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.DropTable(ctx, op.Schema, op.Name)
	}
}

// GetReverse for a DropTable returns a no-op migration. Logically, CreateTable is the reverse,
// but DropTable does not have the table's definition to create one.
//
// TODO: we can fetch table definitions for deleted tables
// from the database engine and execute them as a raw query.
func (op *DropTable) GetReverse() Operation {
	return &noop{}
}

// trimSchema drops schema name from the table name.
// This is a workaroud until schema.Table.Schema is fully integrated with other bun packages.
func trimSchema(name string) string {
	if strings.Contains(name, ".") {
		return strings.Split(name, ".")[1]
	}
	return name
}

type AddForeignKey struct {
	SourceSchema  string
	SourceTable   string
	SourceColumns []string
	TargetSchema  string
	TargetTable   string
	TargetColumns []string
}

var _ Operation = (*AddForeignKey)(nil)

func (op AddForeignKey) String() string {
	return fmt.Sprintf("AddForeignKey %s.%s(%s) references %s.%s(%s)",
		op.SourceSchema, op.SourceTable, strings.Join(op.SourceColumns, ","),
		op.SourceTable, op.TargetTable, strings.Join(op.TargetColumns, ","),
	)
}

func (op *AddForeignKey) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.AddContraint(ctx, sqlschema.FK{
			From: sqlschema.C(op.SourceSchema, op.SourceTable, op.SourceColumns...),
			To:   sqlschema.C(op.TargetSchema, op.TargetTable, op.TargetColumns...),
		}, "dummy_name_"+fmt.Sprint(time.Now().UnixNano()))
	}
}

func (op *AddForeignKey) GetReverse() Operation {
	return &noop{} // TODO: unless the WithFKNameFunc is specified, we cannot know what the constraint is called
}

type DropForeignKey struct {
	Schema         string
	Table          string
	ConstraintName string
}

var _ Operation = (*DropForeignKey)(nil)

func (op *DropForeignKey) String() string {
	return fmt.Sprintf("DropFK %q on table %q.%q", op.ConstraintName, op.Schema, op.Table)
}

func (op *DropForeignKey) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.DropContraint(ctx, op.Schema, op.Table, op.ConstraintName)
	}
}

func (op *DropForeignKey) GetReverse() Operation {
	return &noop{} // TODO: store "OldFK" to recreate it
}

// sqlschema utils ------------------------------------------------------------

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
