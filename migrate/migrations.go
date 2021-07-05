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
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type Command struct {
	Name string
}

type MigrationsOption func(m *Migrations)

func WithTableName(table string) MigrationsOption {
	return func(m *Migrations) {
		m.table = table
	}
}

func WithLocksTableName(table string) MigrationsOption {
	return func(m *Migrations) {
		m.locksTable = table
	}
}

type Migrations struct {
	ms []Migration

	table      string
	locksTable string
}

func NewMigrations(opts ...MigrationsOption) *Migrations {
	m := &Migrations{
		table:      "bun_migrations",
		locksTable: "bun_migration_locks",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Migrations) Migrations() []Migration {
	return m.ms
}

func (m *Migrations) MustRegister(up, down MigrationFunc) {
	if err := m.Register(up, down); err != nil {
		panic(err)
	}
}

func (m *Migrations) Register(up, down MigrationFunc) error {
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

	return nil
}

func (m *Migrations) DiscoverCaller() error {
	dir := filepath.Dir(migrationFile())
	return m.Discover(os.DirFS(dir))
}

func (m *Migrations) Discover(fsys fs.FS) error {
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

func (m *Migrations) getOrCreateMigration(name string) *Migration {
	for i := range m.ms {
		m := &m.ms[i]
		if m.Name == name {
			return m
		}
	}

	m.ms = append(m.ms, Migration{Name: name})
	return &m.ms[len(m.ms)-1]
}

func (m *Migrations) Init(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model((*Migration)(nil)).
		ModelTableExpr(m.table).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}
	if _, err := db.NewCreateTable().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTable).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Migrations) Migrate(ctx context.Context, db *bun.DB) error {
	if len(m.ms) == 0 {
		return errors.New("migrate: there are no any migrations")
	}

	if err := m.Lock(ctx, db); err != nil {
		return err
	}
	defer m.Unlock(ctx, db) //nolint:errcheck

	migrations, lastGroupID, err := m.selectNewMigrations(ctx, db)
	if err != nil {
		return err
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

func (m *Migrations) Rollback(ctx context.Context, db *bun.DB) error {
	if len(m.ms) == 0 {
		return errors.New("migrate: there are no any migrations")
	}

	if err := m.Lock(ctx, db); err != nil {
		return err
	}
	defer m.Unlock(ctx, db) //nolint:errcheck

	lastGroup, lastGroupID, err := m.selectLastGroup(ctx, db)
	if err != nil {
		return err
	}
	if lastGroupID == 0 {
		return errors.New("migrate: there are no any migrations to rollback")
	}

	fmt.Printf("rolling back group #%d with %d migrations...\n", lastGroupID, len(lastGroup))

	for i := range lastGroup {
		if err := m.runDown(ctx, db, &lastGroup[i]); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrations) selectLastGroup(ctx context.Context, db *bun.DB) ([]Migration, int64, error) {
	completed, lastGroupID, err := m.selectCompletedMigrations(ctx, db)
	if err != nil {
		return nil, 0, err
	}
	if lastGroupID == 0 {
		return nil, 0, nil
	}

	var group []Migration

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
			return nil, 0, fmt.Errorf("migrate: can't find migration %q", name)
		}

		migration.ID = id
		group = append(group, *migration)
	}

	return group, lastGroupID, nil
}

func (m *Migrations) MarkCompleted(ctx context.Context, db *bun.DB) error {
	if len(m.ms) == 0 {
		return errors.New("migrate: there are no any migrations")
	}

	if err := m.Lock(ctx, db); err != nil {
		return err
	}
	defer m.Unlock(ctx, db) //nolint:errcheck

	migrations, lastGroupID, err := m.selectNewMigrations(ctx, db)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("nothing to run - database is up to date\n")
		return nil
	}

	groupID := lastGroupID + 1
	fmt.Printf("marking group #%d with %d migrations as completed...\n", groupID, len(migrations))

	for i := range migrations {
		migration := &migrations[i]
		migration.GroupID = groupID
		migration.Up = nil
		if err := m.runUp(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

type Status struct {
	Migrations           []Migration
	NewMigrations        []Migration
	LastMigrationGroup   []Migration
	LastMigrationGroupID int64
}

func (s Status) String() string {
	var sb strings.Builder

	if len(s.Migrations) == 0 {
		sb.WriteString("No migrations available")
		return sb.String()
	}

	sb.WriteString("Total number of migrations: ")
	sb.WriteString(strconv.Itoa(len(s.Migrations)))
	sb.WriteString(" (")
	sb.WriteString(migrationNames(s.Migrations))
	sb.WriteString(")\n")

	if s.LastMigrationGroupID == 0 {
		sb.WriteString("No migrations to rollback")
		return sb.String()
	}

	sb.WriteString("Last migration group: ")
	sb.WriteString(strconv.FormatInt(s.LastMigrationGroupID, 10))
	sb.WriteString(" (")
	sb.WriteString(migrationNames(s.LastMigrationGroup))
	sb.WriteString(")\n")

	if len(s.NewMigrations) == 0 {
		sb.WriteString("Database is up to date")
		return sb.String()
	}

	sb.WriteString("Number of new migrations: ")
	sb.WriteString(strconv.Itoa(len(s.NewMigrations)))
	sb.WriteString(" (")
	sb.WriteString(migrationNames(s.NewMigrations))
	sb.WriteString(")")

	return sb.String()
}

func (m *Migrations) Status(ctx context.Context, db *bun.DB) (*Status, error) {
	status := new(Status)
	status.Migrations = m.sortedMigrations()
	if len(status.Migrations) == 0 {
		return status, nil
	}

	lastGroup, lastGroupID, err := m.selectLastGroup(ctx, db)
	if err != nil {
		return nil, err
	}

	status.LastMigrationGroup = lastGroup
	status.LastMigrationGroupID = lastGroupID

	newMigrations, _, err := m.selectNewMigrations(ctx, db)
	if err != nil {
		return nil, err
	}

	status.NewMigrations = newMigrations

	return status, nil
}

func migrationNames(migrations []Migration) string {
	if len(migrations) > 5 {
		return migrations[0].Name + " ... " + migrations[len(migrations)-1].Name
	}

	var sb strings.Builder
	for i := range migrations {
		sb.WriteString(migrations[i].Name)
		if i+1 != len(migrations) {
			sb.WriteString(", ")
		}
	}

	return sb.String()
}

func (m *Migrations) CreateGo(ctx context.Context, db *bun.DB, name string) error {
	name, err := m.genMigrationName(name)
	if err != nil {
		return err
	}

	fname := name + ".go"
	fpath := filepath.Join(migrationsDir(), fname)

	fmt.Printf("creating %s...\n", fname)
	return ioutil.WriteFile(fpath, []byte(goTemplate), 0o644)
}

func (m *Migrations) CreateSQL(ctx context.Context, db *bun.DB, name string) error {
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

func (m *Migrations) genMigrationName(name string) (string, error) {
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

func (m *Migrations) runUp(ctx context.Context, db *bun.DB, migration *Migration) error {
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

func (m *Migrations) runDown(ctx context.Context, db *bun.DB, migration *Migration) error {
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
		Where("id = ?", migration.ID).
		Exec(ctx)
	return err
}

// selectCompletedMigrations selects completed migrations in descending order
// (the order is used for rollbacks).
func (m *Migrations) selectCompletedMigrations(
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

func (m *Migrations) selectNewMigrations(
	ctx context.Context, db *bun.DB,
) ([]Migration, int64, error) {
	migrations := m.sortedMigrations()

	completed, lastGroupID, err := m.selectCompletedMigrations(ctx, db)
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

func (m *Migrations) sortedMigrations() []Migration {
	migrations := make([]Migration, len(m.ms))
	copy(migrations, m.ms)

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations
}

func (m *Migrations) formattedTableName(db *bun.DB) string {
	return db.Formatter().FormatQuery(m.table)
}

func (m *Migrations) tableNameWithAlias() string {
	return m.table + " AS m"
}

func (m *Migrations) locksTableNameWithAlias() string {
	return m.locksTable + " AS l"
}

//------------------------------------------------------------------------------

type migrationLock struct {
	ID        int64  `bun:"alias:l"`
	TableName string `bun:",unique"`
}

func (m *Migrations) Lock(ctx context.Context, db *bun.DB) error {
	lock := &migrationLock{
		TableName: m.formattedTableName(db),
	}
	if _, err := db.NewInsert().
		Model(lock).
		ModelTableExpr(m.locksTableNameWithAlias()).
		Exec(ctx); err != nil {

		return fmt.Errorf("bun: migrations table is already locked (%w)", err)
	}
	return nil
}

func (m *Migrations) Unlock(ctx context.Context, db *bun.DB) error {
	tableName := m.formattedTableName(db)
	_, err := db.NewDelete().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTableNameWithAlias()).
		Where("? = ?", bun.Ident("table_name"), tableName).
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
		if !strings.Contains(f.Function, "/bun/migrate.") {
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
