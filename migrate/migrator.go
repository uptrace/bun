package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type Command struct {
	Name string
}

type MigratorOption func(m *Migrator)

func WithAutoDiscover() MigratorOption {
	return func(m *Migrator) {
		m.autoDiscover = true
	}
}

func WithTableName(table string) MigratorOption {
	return func(m *Migrator) {
		m.table = table
	}
}

type Migrator struct {
	ms []Migration

	table string

	autoDiscover   bool
	discoveredDirs map[string]struct{}
}

func NewMigrator(opts ...MigratorOption) *Migrator {
	m := &Migrator{
		table: "migrations",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Migrator) init() error {
	if m.autoDiscover {
		if err := m.autoDiscoverFile(migrationFile()); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) MustRegister(up, down MigrationFunc) {
	if err := m.Register(up, down); err != nil {
		panic(err)
	}
}

func (m *Migrator) Register(up, down MigrationFunc) error {
	fpath := migrationFile()
	name, err := extractMigrationName(fpath)
	if err != nil {
		return err
	}

	m.ms = append(m.ms, Migration{
		Name: name,
		Up:   up,
		Down: down,
	})

	if m.autoDiscover {
		return m.autoDiscoverFile(fpath)
	}
	return nil
}

func (m *Migrator) autoDiscoverFile(fpath string) error {
	fpath, err := filepath.Abs(fpath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(fpath)

	if _, ok := m.discoveredDirs[dir]; ok {
		return nil
	}

	if m.discoveredDirs == nil {
		m.discoveredDirs = make(map[string]struct{})
	}
	m.discoveredDirs[dir] = struct{}{}

	return m.Discover(os.DirFS(dir))
}

func (m *Migrator) Discover(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".up.sql") && !strings.HasSuffix(path, ".down.sql") {
			return nil
		}

		name, err := extractMigrationName(path)
		if err != nil {
			return err
		}

		migration := m.getOrCreateMigration(name)
		if err != nil {
			return err
		}
		migrationFunc := NewSQLMigrationFunc(fsys, path)

		if strings.HasSuffix(path, ".up.sql") {
			migration.Up = migrationFunc
			return nil
		}
		if strings.HasSuffix(path, ".down.sql") {
			migration.Down = migrationFunc
			return nil
		}

		return errors.New("not reached")
	})
}

func (m *Migrator) getOrCreateMigration(name string) *Migration {
	for i := range m.ms {
		m := &m.ms[i]
		if m.Name == name {
			return m
		}
	}

	m.ms = append(m.ms, Migration{Name: name})
	return &m.ms[len(m.ms)-1]
}

func (m *Migrator) Init(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*Migration)(nil),
		(*migrationLock)(nil),
	}
	for _, model := range models {
		if _, err := db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) Migrate(ctx context.Context, db *bun.DB) error {
	if err := m.init(); err != nil {
		return err
	}
	if len(m.ms) == 0 {
		return errors.New("there are no any migrations")
	}

	migrations := make([]Migration, len(m.ms))
	copy(migrations, m.ms)

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	if err := m.Lock(ctx, db); err != nil {
		return err
	}
	defer m.Unlock(ctx, db) //nolint:errcheck

	completed, lastGroupID, err := m.selectMigrations(ctx, db)
	if err != nil {
		return err
	}

	completedMap := migrationMap(completed)
	for i := len(migrations) - 1; i >= 0; i-- {
		migration := &migrations[i]
		if _, ok := completedMap[migration.Name]; ok {
			migrations = append(migrations[:i], migrations[i+1:]...)
		}
	}

	if len(migrations) == 0 {
		fmt.Printf("nothing to run - database is up to date\n")
		return nil
	}

	groupID := lastGroupID + 1
	fmt.Printf("running group #%d with %d migrations...\n", groupID, len(migrations))

	for i := range migrations {
		migration := &migrations[i]
		migration.GroupID = groupID
		if err := m.runUp(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) Rollback(ctx context.Context, db *bun.DB) error {
	if err := m.init(); err != nil {
		return err
	}
	if len(m.ms) == 0 {
		return errors.New("there are no any migrations")
	}

	if err := m.Lock(ctx, db); err != nil {
		return err
	}
	defer m.Unlock(ctx, db) //nolint:errcheck

	completed, lastGroupID, err := m.selectMigrations(ctx, db)
	if err != nil {
		return err
	}

	if lastGroupID == 0 {
		return errors.New("there are no any migrations to rollback")
	}

	var group []*Migration

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
			return fmt.Errorf("can't find migration %q", name)
		}

		migration.ID = id
		group = append(group, migration)
	}

	fmt.Printf("rolling back group #%d with %d migrations...\n", lastGroupID, len(group))

	for _, migration := range group {
		if err := m.runDown(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) CreateGo(ctx context.Context, db *bun.DB, name string) error {
	name, err := m.genMigrationName(name)
	if err != nil {
		return err
	}

	fname := name + ".go"
	fpath := filepath.Join(migrationsDir(), fname)

	fmt.Printf("creating %s...\n", fname)
	return ioutil.WriteFile(fpath, []byte(goTemplate), 0o644)
}

func (m *Migrator) CreateSQL(ctx context.Context, db *bun.DB, name string) error {
	name, err := m.genMigrationName(name)
	if err != nil {
		return err
	}

	fname := name + ".up.sql"
	fpath := filepath.Join(migrationsDir(), fname)

	fmt.Printf("creating %s...\n", fname)
	return ioutil.WriteFile(fpath, []byte(sqlTemplate), 0o644)
}

var nameRE = regexp.MustCompile(`^[0-9a-z_\-]+$`)

func (m *Migrator) genMigrationName(name string) (string, error) {
	const timeFormat = "20060102150405"

	if name == "" {
		return "", errors.New("create requires a migration name")
	}
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("invalid migration name: %q", name)
	}

	version := time.Now().UTC().Format(timeFormat)
	return fmt.Sprintf("%s_%s", version, name), nil
}

func (m *Migrator) runUp(ctx context.Context, db *bun.DB, migration *Migration) error {
	fmt.Printf("\trunning migration %s... ", migration.Name)
	if migration.Up != nil {
		if err := migration.Up(ctx, db); err != nil {
			fmt.Printf("%s\n", err)
			return err
		} else {
			fmt.Printf("OK\n")
		}
	} else {
		fmt.Printf("nothing to run\n")
	}

	_, err := db.NewInsert().Model(migration).
		ModelTableExpr(m.tableNameWithAlias()).
		Exec(ctx)
	return err
}

func (m *Migrator) runDown(ctx context.Context, db *bun.DB, migration *Migration) error {
	fmt.Printf("\trolling back migration %s... ", migration.Name)
	if migration.Down != nil {
		if err := migration.Down(ctx, db); err != nil {
			fmt.Printf("%s\n", err)
			return err
		} else {
			fmt.Printf("OK\n")
		}
	} else {
		fmt.Printf("nothing to run\n")
	}

	_, err := db.NewDelete().
		Model(migration).
		ModelTableExpr(m.tableNameWithAlias()).
		WherePK().
		Exec(ctx)
	return err
}

func (m *Migrator) selectMigrations(
	ctx context.Context, db *bun.DB,
) ([]Migration, int64, error) {
	var ms []Migration
	if err := db.NewSelect().
		Model(&ms).
		ModelTableExpr(m.tableNameWithAlias()).
		OrderExpr("m.id DESC").
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

func (m *Migrator) tableNameWithAlias() string {
	return m.table + " AS m"
}

//------------------------------------------------------------------------------

type migrationLock struct {
	ID        int64
	TableName string `bun:",unique"`
}

func (m *Migrator) Lock(ctx context.Context, db *bun.DB) error {
	lock := &migrationLock{
		TableName: m.table,
	}
	if _, err := db.NewInsert().Model(lock).Exec(ctx); err != nil {
		return fmt.Errorf("bun: migrations table is already locked (%w)", err)
	}
	return nil
}

func (m *Migrator) Unlock(ctx context.Context, db *bun.DB) error {
	_, err := db.NewDelete().Model((*migrationLock)(nil)).
		Where("? = ?", bun.Ident("table_name"), m.table).
		Exec(ctx)
	return err
}

func migrationMap(ms []Migration) map[string]*Migration {
	mp := make(map[string]*Migration)
	for i := range ms {
		m := &ms[i]
		mp[m.Name] = m
	}
	return mp
}

func migrationFile() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		f, ok := frames.Next()
		if !ok {
			break
		}
		if !strings.Contains(f.Function, "/bun/migrate") {
			return f.File
		}
	}

	return ""
}

func migrationsDir() string {
	return filepath.Dir(migrationFile())
}

var fnameRE = regexp.MustCompile(`^(\d{14})_[0-9a-z_\-]+\.`)

func extractMigrationName(fpath string) (string, error) {
	fname := filepath.Base(fpath)

	matches := fnameRE.FindStringSubmatch(fname)
	if matches == nil {
		return "", fmt.Errorf("unsupported migration name format: %q", fname)
	}

	return matches[1], nil
}
