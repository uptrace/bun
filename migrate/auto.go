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
		m.diffOpts = append(m.diffOpts, FKNameFunc(f))
	}
}

// WithRenameFK prevents AutoMigrator from recreating foreign keys when their dependent relations are renamed,
// and forces it to run a RENAME CONSTRAINT query instead. Creating an index on a large table can take a very long time,
// and in those cases simply renaming the FK makes a lot more sense.
func WithRenameFK(enabled bool) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.diffOpts = append(m.diffOpts, DetectRenamedFKs(enabled))
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

	// diffOpts are passed to Diff.
	diffOpts []DiffOption

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
	return Diff(got, want, am.diffOpts...), nil
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
// TODO: move to migrate/internal

type DiffOption func(*detectorConfig)

func FKNameFunc(f func(sqlschema.FK) string) DiffOption {
	return func(cfg *detectorConfig) {
		cfg.FKNameFunc = f
	}
}

func DetectRenamedFKs(enabled bool) DiffOption {
	return func(cfg *detectorConfig) {
		cfg.DetectRenamedFKs = enabled
	}
}

func Diff(got, want sqlschema.State, opts ...DiffOption) Changeset {
	detector := newDetector(got, want, opts...)
	return detector.DetectChanges()
}

// detectorConfig controls how differences in the model states are resolved.
type detectorConfig struct {
	FKNameFunc       func(sqlschema.FK) string
	DetectRenamedFKs bool
}

type detector struct {
	// current state represents the existing database schema.
	current sqlschema.State

	// target state represents the database schema defined in bun models.
	target sqlschema.State

	changes Changeset
	refMap  sqlschema.RefMap

	// fkNameFunc builds the name for created/renamed FK contraints.
	fkNameFunc func(sqlschema.FK) string

	// detectRenemedFKS controls how FKs are treated when their references (table/column) are renamed.
	detectRenamedFKs bool
}

func newDetector(got, want sqlschema.State, opts ...DiffOption) *detector {
	cfg := &detectorConfig{
		FKNameFunc:       defaultFKName,
		DetectRenamedFKs: false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var existingFKs []sqlschema.FK
	for fk := range got.FKs {
		existingFKs = append(existingFKs, fk)
	}

	return &detector{
		current:          got,
		target:           want,
		refMap:           sqlschema.NewRefMap(existingFKs...),
		fkNameFunc:       cfg.FKNameFunc,
		detectRenamedFKs: cfg.DetectRenamedFKs,
	}
}

func (d *detector) DetectChanges() Changeset {
	// Discover CREATE/RENAME/DROP TABLE
	targetTables := newTableSet(d.target.Tables...)
	currentTables := newTableSet(d.current.Tables...) // keeps state (which models still need to be checked)

	// These table sets record "updates" to the targetTables set.
	created := newTableSet()
	renamed := newTableSet()

	addedTables := targetTables.Sub(currentTables)
AddedLoop:
	for _, added := range addedTables.Values() {
		removedTables := currentTables.Sub(targetTables)
		for _, removed := range removedTables.Values() {
			if d.canRename(removed, added) {
				d.changes.Add(&RenameTable{
					Schema: removed.Schema,
					From:   removed.Name,
					To:     added.Name,
				})

				d.detectRenamedColumns(removed, added)

				// Update referenced table in all related FKs
				if d.detectRenamedFKs {
					d.refMap.UpdateT(removed.T(), added.T())
				}

				renamed.Add(added)

				// Do not check this model further, we know it was renamed.
				currentTables.Remove(removed.Name)
				continue AddedLoop
			}
		}
		// If a new table did not appear because of the rename operation, then it must've been created.
		d.changes.Add(&CreateTable{
			Schema: added.Schema,
			Name:   added.Name,
			Model:  added.Model,
		})
		created.Add(added)
	}

	// Tables that aren't present anymore and weren't renamed or left untouched were deleted.
	dropped := currentTables.Sub(targetTables)
	for _, t := range dropped.Values() {
		d.changes.Add(&DropTable{
			Schema: t.Schema,
			Name:   t.Name,
		})
	}

	// Detect changes in existing tables that weren't renamed
	// TODO: here having State.Tables be a map[string]Table would be much more convenient.
	// Then we can alse retire tableSet, or at least simplify it to a certain extent.
	curEx := currentTables.Sub(dropped)
	tarEx := targetTables.Sub(created).Sub(renamed)
	for _, target := range tarEx.Values() {
		// This step is redundant if we have map[string]Table
		var current sqlschema.Table
		for _, cur := range curEx.Values() {
			if cur.Name == target.Name {
				current = cur
				break
			}
		}
		d.detectRenamedColumns(current, target)
	}

	// Compare and update FKs ----------------
	currentFKs := make(map[sqlschema.FK]string)
	for k, v := range d.current.FKs {
		currentFKs[k] = v
	}

	if d.detectRenamedFKs {
		// Add RenameFK migrations for updated FKs.
		for old, renamed := range d.refMap.Updated() {
			newName := d.fkNameFunc(renamed)
			d.changes.Add(&RenameFK{
				FK:   renamed, // TODO: make sure this is applied after the table/columns are renamed
				From: d.current.FKs[old],
				To:   d.fkNameFunc(renamed),
			})

			// Here we can add this fk to "current.FKs" to prevent it from firing in the next 2 for-loops.
			currentFKs[renamed] = newName
			delete(currentFKs, old)
		}
	}

	// Add AddFK migrations for newly added FKs.
	for fk := range d.target.FKs {
		if _, ok := currentFKs[fk]; !ok {
			d.changes.Add(&AddFK{
				FK:             fk,
				ConstraintName: d.fkNameFunc(fk),
			})
		}
	}

	// Add DropFK migrations for removed FKs.
	for fk, fkName := range currentFKs {
		if _, ok := d.target.FKs[fk]; !ok {
			d.changes.Add(&DropFK{
				FK:             fk,
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

func (d *detector) detectRenamedColumns(removed, added sqlschema.Table) {
	for aName, aCol := range added.Columns {
		// This column exists in the database, so it wasn't renamed
		if _, ok := removed.Columns[aName]; ok {
			continue
		}
		for rName, rCol := range removed.Columns {
			if aCol != rCol {
				continue
			}
			d.changes.Add(&RenameColumn{
				Schema: added.Schema,
				Table:  added.Name,
				From:   rName,
				To:     aName,
			})
			delete(removed.Columns, rName) // no need to check this column again
			d.refMap.UpdateC(sqlschema.C(added.Schema, added.Name, rName), aName)
		}
	}
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
		Schema: op.Schema,
		From:   op.To,
		To:     op.From,
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

// defaultFKName returns a name for the FK constraint in the format {tablename}_{columnname(s)}_fkey, following the Postgres convention.
func defaultFKName(fk sqlschema.FK) string {
	columnnames := strings.Join(fk.From.Column.Split(), "_")
	return fmt.Sprintf("%s_%s_fkey", fk.From.Table, columnnames)
}

type AddFK struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*AddFK)(nil)

func (op AddFK) String() string {
	source, target := op.FK.From, op.FK.To
	return fmt.Sprintf("AddForeignKey %q %s.%s(%s) references %s.%s(%s)", op.ConstraintName,
		source.Schema, source.Table, strings.Join(source.Column.Split(), ","),
		target.Schema, target.Table, strings.Join(target.Column.Split(), ","),
	)
}

func (op *AddFK) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.AddContraint(ctx, op.FK, op.ConstraintName)
	}
}

func (op *AddFK) GetReverse() Operation {
	return &DropFK{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

type DropFK struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*DropFK)(nil)

func (op *DropFK) String() string {
	source := op.FK.From.T()
	return fmt.Sprintf("DropFK %q on table %q.%q", op.ConstraintName, source.Schema, source.Table)
}

func (op *DropFK) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		source := op.FK.From.T()
		return m.DropContraint(ctx, source.Schema, source.Table, op.ConstraintName)
	}
}

func (op *DropFK) GetReverse() Operation {
	return &AddFK{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

// RenameFK
type RenameFK struct {
	FK   sqlschema.FK
	From string
	To   string
}

var _ Operation = (*RenameFK)(nil)

func (op *RenameFK) String() string {
	return "RenameFK"
}

func (op *RenameFK) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		table := op.FK.From
		return m.RenameConstraint(ctx, table.Schema, table.Table, op.From, op.To)
	}
}

func (op *RenameFK) GetReverse() Operation {
	return &RenameFK{
		FK:   op.FK,
		From: op.From,
		To:   op.To,
	}
}

// RenameColumn
type RenameColumn struct {
	Schema string
	Table  string
	From   string
	To     string
}

var _ Operation = (*RenameColumn)(nil)

func (op RenameColumn) String() string {
	return ""
}

func (op *RenameColumn) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return m.RenameColumn(ctx, op.Schema, op.Table, op.From, op.To)
	}
}

func (op *RenameColumn) GetReverse() Operation {
	return &RenameColumn{
		Schema: op.Schema,
		Table:  op.Table,
		From:   op.To,
		To:     op.From,
	}
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
