package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
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

func (m *Migrator) Migrations() *Migrations {
	return m.migrations
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

func (m *Migrator) Migrate(ctx context.Context) (*MigrationGroup, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}

	if err := m.Lock(ctx); err != nil {
		return nil, err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	migrations, lastGroupID, err := m.selectNewMigrations(ctx)
	if err != nil {
		return nil, err
	}

	group := &MigrationGroup{
		Migrations: migrations,
	}

	if len(migrations) == 0 {
		return group, nil
	}

	group.ID = lastGroupID + 1

	for i := range migrations {
		migration := &migrations[i]
		migration.GroupID = group.ID
		if err := m.runUp(ctx, m.db, migration); err != nil {
			return nil, err
		}
	}

	return group, nil
}

func (m *Migrator) Rollback(ctx context.Context) (*MigrationGroup, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}

	if err := m.Lock(ctx); err != nil {
		return nil, err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	lastGroup, err := m.selectLastGroup(ctx)
	if err != nil {
		return nil, err
	}

	for i := range lastGroup.Migrations {
		if err := m.runDown(ctx, m.db, &lastGroup.Migrations[i]); err != nil {
			return nil, err
		}
	}

	return lastGroup, nil
}

func (m *Migrator) selectLastGroup(ctx context.Context) (*MigrationGroup, error) {
	completed, lastGroupID, err := m.selectCompletedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	group := &MigrationGroup{
		ID: lastGroupID,
	}
	if group.ID == 0 {
		return group, nil
	}

	migrationMap := migrationMap(m.ms)
	for i := range completed {
		migration := &completed[i]
		if migration.GroupID != lastGroupID {
			continue
		}

		id := migration.ID
		name := migration.Name

		migration, ok := migrationMap[name]
		if !ok {
			return nil, fmt.Errorf("migrate: can't find migration %q", name)
		}

		migration.ID = id
		group.Migrations = append(group.Migrations, *migration)
	}

	return group, nil
}

func (m *Migrator) MarkCompleted(ctx context.Context) (*MigrationGroup, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}

	if err := m.Lock(ctx); err != nil {
		return nil, err
	}
	defer m.Unlock(ctx) //nolint:errcheck

	migrations, lastGroupID, err := m.selectNewMigrations(ctx)
	if err != nil {
		return nil, err
	}

	if len(migrations) == 0 {
		return new(MigrationGroup), nil
	}

	group := &MigrationGroup{
		ID:         lastGroupID + 1,
		Migrations: migrations,
	}

	for i := range migrations {
		migration := &migrations[i]
		migration.GroupID = group.ID
		migration.Up = nil
		if err := m.runUp(ctx, m.db, migration); err != nil {
			return nil, err
		}
	}

	return group, nil
}

type MigrationStatus struct {
	Migrations    MigrationSlice
	NewMigrations MigrationSlice
	LastGroup     *MigrationGroup
}

func (m *Migrator) Status(ctx context.Context) (*MigrationStatus, error) {
	status := new(MigrationStatus)
	status.Migrations = m.migrations.Sorted()

	lastGroup, err := m.selectLastGroup(ctx)
	if err != nil {
		return nil, err
	}
	status.LastGroup = lastGroup

	newMigrations, _, err := m.selectNewMigrations(ctx)
	if err != nil {
		return nil, err
	}
	status.NewMigrations = newMigrations

	return status, nil
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

func (m *Migrator) runUp(ctx context.Context, db *bun.DB, migration *Migration) error {
	if migration.Up != nil {
		if err := migration.Up(ctx, db); err != nil {
			return err
		}
	}

	_, err := db.NewInsert().Model(migration).
		ModelTableExpr(m.table).
		Exec(ctx)
	return err
}

func (m *Migrator) runDown(ctx context.Context, db *bun.DB, migration *Migration) error {
	if migration.Down != nil {
		if err := migration.Down(ctx, db); err != nil {
			return err
		}
	}

	_, err := db.NewDelete().
		Model(migration).
		ModelTableExpr(m.table).
		Where("id = ?", migration.ID).
		Exec(ctx)
	return err
}

// selectCompletedMigrations selects completed migrations in descending order
// (the order is used for rollbacks).
func (m *Migrator) selectCompletedMigrations(ctx context.Context) (MigrationSlice, int64, error) {
	var ms MigrationSlice
	if err := m.db.NewSelect().
		ColumnExpr("*").
		Model(&ms).
		ModelTableExpr(m.table).
		OrderExpr("id DESC").
		Scan(ctx); err != nil {
		return nil, 0, err
	}

	var lastGroupID int64

	for i := range ms {
		groupID := ms[i].GroupID
		if groupID > lastGroupID {
			lastGroupID = groupID
		}
	}

	return ms, lastGroupID, nil
}

func (m *Migrator) selectNewMigrations(ctx context.Context) (MigrationSlice, int64, error) {
	migrations := m.migrations.Sorted()

	completed, lastGroupID, err := m.selectCompletedMigrations(ctx)
	if err != nil {
		return nil, 0, err
	}

	completedMap := migrationMap(completed)
	for i := len(migrations) - 1; i >= 0; i-- {
		migration := &migrations[i]
		if _, ok := completedMap[migration.Name]; ok {
			migrations = append(migrations[:i], migrations[i+1:]...)
		}
	}

	return migrations, lastGroupID, nil
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
