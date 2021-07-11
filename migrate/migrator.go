package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"time"

	"github.com/uptrace/bun"
)

type MigratorOption func(m *Migrator)

func WithTableName(table string) MigratorOption {
	return func(m *Migrator) {
		m.table = table
	}
}

func WithLocksTableName(table string) MigratorOption {
	return func(m *Migrator) {
		m.locksTable = table
	}
}

type Migrator struct {
	db         *bun.DB
	migrations *Migrations

	ms MigrationSlice

	table      string
	locksTable string
}

func NewMigrator(db *bun.DB, migrations *Migrations, opts ...MigratorOption) *Migrator {
	m := &Migrator{
		db:         db,
		migrations: migrations,

		ms: migrations.ms,

		table:      "bun_migrations",
		locksTable: "bun_migration_locks",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Migrator) DB() *bun.DB {
	return m.db
}

// MigrationsWithStatus returns migrations with status in ascending order.
func (m *Migrator) MigrationsWithStatus(ctx context.Context) (MigrationSlice, error) {
	sorted := m.migrations.Sorted()

	applied, err := m.selectAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := migrationMap(applied)
	for i := range sorted {
		name := sorted[i].Name
		if m2, ok := appliedMap[name]; ok {
			sorted[i] = *m2
		}
	}

	return sorted, nil
}

func (m *Migrator) Init(ctx context.Context) error {
	if _, err := m.db.NewCreateTable().
		Model((*Migration)(nil)).
		ModelTableExpr(m.table).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}
	if _, err := m.db.NewCreateTable().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTable).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) Migrate(ctx context.Context, opts ...MigrationOption) (*MigrationGroup, error) {
	cfg := newMigrationConfig(opts)

	if err := m.validate(); err != nil {
		return nil, err
	}

	if err := m.Lock(ctx); err != nil {
		return nil, err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	migrations, err := m.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, err
	}

	group := &MigrationGroup{
		Migrations: migrations.Unapplied(),
	}
	if len(group.Migrations) == 0 {
		return group, nil
	}
	group.ID = migrations.LastGroupID() + 1

	for i := range group.Migrations {
		migration := &group.Migrations[i]
		migration.GroupID = group.ID

		if !cfg.withoutMigrationFunc && migration.Up != nil {
			if err := migration.Up(ctx, m.db); err != nil {
				return nil, err
			}
		}

		if err := m.MarkApplied(ctx, migration); err != nil {
			return nil, err
		}
	}

	return group, nil
}

func (m *Migrator) Rollback(ctx context.Context, opts ...MigrationOption) (*MigrationGroup, error) {
	cfg := newMigrationConfig(opts)

	if err := m.validate(); err != nil {
		return nil, err
	}

	if err := m.Lock(ctx); err != nil {
		return nil, err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	migrations, err := m.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, err
	}

	lastGroup := migrations.LastGroup()

	for i := range lastGroup.Migrations {
		migration := &lastGroup.Migrations[i]

		if !cfg.withoutMigrationFunc && migration.Down != nil {
			if err := migration.Down(ctx, m.db); err != nil {
				return nil, err
			}
		}

		if err := m.MarkUnapplied(ctx, migration); err != nil {
			return nil, err
		}
	}

	return lastGroup, nil
}

type MigrationStatus struct {
	Migrations    MigrationSlice
	NewMigrations MigrationSlice
	LastGroup     *MigrationGroup
}

func (m *Migrator) Status(ctx context.Context) (*MigrationStatus, error) {
	log.Printf(
		"DEPRECATED: bun: replace Status(ctx) with " +
			"MigrationsWithStatus(ctx)")

	migrations, err := m.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &MigrationStatus{
		Migrations:    migrations,
		NewMigrations: migrations.Unapplied(),
		LastGroup:     migrations.LastGroup(),
	}, nil
}

func (m *Migrator) MarkCompleted(ctx context.Context) (*MigrationGroup, error) {
	log.Printf(
		"DEPRECATED: bun: replace MarkCompleted(ctx) with " +
			"Migrate(ctx, migrate.WithoutMigrationFunc())")

	return m.Migrate(ctx, WithoutMigrationFunc())
}

func (m *Migrator) CreateGo(ctx context.Context, name string) (*MigrationFile, error) {
	name, err := m.genMigrationName(name)
	if err != nil {
		return nil, err
	}

	fname := name + ".go"
	fpath := filepath.Join(m.migrations.getDirectory(), fname)

	if err := ioutil.WriteFile(fpath, []byte(goTemplate), 0o644); err != nil {
		return nil, err
	}

	mf := &MigrationFile{
		FileName: fname,
		FilePath: fpath,
		Content:  goTemplate,
	}
	return mf, nil
}

func (m *Migrator) CreateSQL(ctx context.Context, name string) (*MigrationFile, error) {
	name, err := m.genMigrationName(name)
	if err != nil {
		return nil, err
	}

	fname := name + ".up.sql"
	fpath := filepath.Join(m.migrations.getDirectory(), fname)

	if err := ioutil.WriteFile(fpath, []byte(sqlTemplate), 0o644); err != nil {
		return nil, err
	}

	mf := &MigrationFile{
		FileName: fname,
		FilePath: fpath,
		Content:  goTemplate,
	}
	return mf, nil
}

var nameRE = regexp.MustCompile(`^[0-9a-z_\-]+$`)

func (m *Migrator) genMigrationName(name string) (string, error) {
	const timeFormat = "20060102150405"

	if name == "" {
		return "", errors.New("migrate: create requires a migration name")
	}
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("migrate: invalid migration name: %q", name)
	}

	version := time.Now().UTC().Format(timeFormat)
	return fmt.Sprintf("%s_%s", version, name), nil
}

// MarkApplied marks the migration as applied (applied).
func (m *Migrator) MarkApplied(ctx context.Context, migration *Migration) error {
	_, err := m.db.NewInsert().Model(migration).
		ModelTableExpr(m.table).
		Exec(ctx)
	return err
}

// MarkUnapplied marks the migration as unapplied (new).
func (m *Migrator) MarkUnapplied(ctx context.Context, migration *Migration) error {
	_, err := m.db.NewDelete().
		Model(migration).
		ModelTableExpr(m.table).
		Where("id = ?", migration.ID).
		Exec(ctx)
	return err
}

// selectAppliedMigrations selects applied (applied) migrations in descending order.
func (m *Migrator) selectAppliedMigrations(ctx context.Context) (MigrationSlice, error) {
	var ms MigrationSlice
	if err := m.db.NewSelect().
		ColumnExpr("*").
		Model(&ms).
		ModelTableExpr(m.table).
		Scan(ctx); err != nil {
		return nil, err
	}
	return ms, nil
}

func (m *Migrator) formattedTableName(db *bun.DB) string {
	return db.Formatter().FormatQuery(m.table)
}

func (m *Migrator) validate() error {
	if len(m.ms) == 0 {
		return errors.New("migrate: there are no any migrations")
	}
	return nil
}

//------------------------------------------------------------------------------

type migrationLock struct {
	ID        int64
	TableName string `bun:",unique"`
}

func (m *Migrator) Lock(ctx context.Context) error {
	lock := &migrationLock{
		TableName: m.formattedTableName(m.db),
	}
	if _, err := m.db.NewInsert().
		Model(lock).
		ModelTableExpr(m.locksTable).
		Exec(ctx); err != nil {
		return fmt.Errorf("migrate: migrations table is already locked (%w)", err)
	}
	return nil
}

func (m *Migrator) Unlock(ctx context.Context) error {
	tableName := m.formattedTableName(m.db)
	_, err := m.db.NewDelete().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTable).
		Where("? = ?", bun.Ident("table_name"), tableName).
		Exec(ctx)
	return err
}

func migrationMap(ms MigrationSlice) map[string]*Migration {
	mp := make(map[string]*Migration)
	for i := range ms {
		m := &ms[i]
		mp[m.Name] = m
	}
	return mp
}
