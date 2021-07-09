package migrate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type MigrationsOption func(m *Migrations)

func WithMigrationsDirectory(directory string) MigrationsOption {
	return func(m *Migrations) {
		m.directory = directory
	}
}

type Migrations struct {
	ms MigrationSlice

	directory string
}

func NewMigrations(opts ...MigrationsOption) *Migrations {
	m := &Migrations{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Migrations) Migrations() MigrationSlice {
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

		return errors.New("migrate: not reached")
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

func (m *Migrations) getDirectory() string {
	if m.directory == "" {
		return filepath.Dir(migrationFile())
	}
	return m.directory
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

var fnameRE = regexp.MustCompile(`^(\d{14})_[0-9a-z_\-]+\.`)

func extractMigrationName(fpath string) (string, error) {
	fname := filepath.Base(fpath)

	matches := fnameRE.FindStringSubmatch(fname)
	if matches == nil {
		return "", fmt.Errorf("migrate: unsupported migration name format: %q", fname)
	}

	return matches[1], nil
}
