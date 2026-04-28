package migrate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/uptrace/bun"
)

// Migration represents a single database migration with up and down functions.
type Migration struct {
	bun.BaseModel

	ID         int64 `bun:",pk,autoincrement"`
	Name       string
	Comment    string `bun:"-"`
	GroupID    int64
	MigratedAt time.Time `bun:",notnull,nullzero,default:current_timestamp"`

	Up   MigrationFunc `bun:"-"`
	Down MigrationFunc `bun:"-"`
}

// String returns the migration name and comment.
func (m Migration) String() string {
	return fmt.Sprintf("%s_%s", m.Name, m.Comment)
}

// IsApplied reports whether the migration has been applied.
func (m Migration) IsApplied() bool {
	return m.ID > 0
}

// MigrationFunc is a function that executes a migration against a database.
type MigrationFunc func(ctx context.Context, db *bun.DB) error

func newSQLMigrationFunc(fsys fs.FS, name string) (MigrationFunc, error) {
	sqlFile, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}

	contents, err := io.ReadAll(sqlFile)
	if err != nil {
		return nil, err
	}

	tmpl := sync.OnceValues(func() (*template.Template, error) {
		return template.New(name).Parse(string(contents))
	})

	return func(ctx context.Context, db *bun.DB) error {
		var reader io.Reader = bytes.NewReader(contents)

		if data := ctx.Value(templateDataKey); data != nil {
			buf, err := renderTemplate(tmpl, data)
			if err != nil {
				return err
			}
			reader = buf
		}

		scanner := bufio.NewScanner(reader)
		var queries []string

		var query []byte
		for scanner.Scan() {
			b := scanner.Bytes()

			const prefix = "--bun:"
			if bytes.HasPrefix(b, []byte(prefix)) {
				b = b[len(prefix):]
				if bytes.Equal(b, []byte("split")) {
					if len(query) > 0 {
						queries = append(queries, string(query))
						query = query[:0]
					}
					continue
				}
				return fmt.Errorf("bun: unknown directive: %q", b)
			}

			if len(bytes.TrimSpace(b)) > 0 {
				query = append(query, b...)
				query = append(query, '\n')
			}
		}

		if len(query) > 0 {
			queries = append(queries, string(query))
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		var idb bun.IConn

		isTx := strings.HasSuffix(name, ".tx.up.sql") ||
			strings.HasSuffix(name, ".tx.down.sql")
		if isTx {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			idb = tx
		} else {
			conn, err := db.Conn(ctx)
			if err != nil {
				return err
			}
			idb = conn
		}

		var retErr error
		var execErr error

		defer func() {
			if tx, ok := idb.(bun.Tx); ok {
				if execErr != nil {
					retErr = tx.Rollback()
				} else {
					retErr = tx.Commit()
				}
				return
			}

			if conn, ok := idb.(bun.Conn); ok {
				retErr = conn.Close()
				return
			}

			panic("not reached")
		}()

		for _, query := range queries {
			if strings.TrimSpace(query) == "" {
				continue
			}
			if _, execErr = db.ExecContext(ctx, query); execErr != nil {
				return execErr
			}
		}

		return retErr
	}, nil
}

//------------------------------------------------------------------------------

type contextKey struct{}

var templateDataKey = contextKey{}

func renderTemplate(parseFunc func() (*template.Template, error), templateData any) (*bytes.Buffer, error) {
	tmpl, err := parseFunc()
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, templateData); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return &rendered, nil
}

const goTemplate = `package %s

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] ")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] ")
		return nil
	})
}
`

const sqlTemplate = `SET statement_timeout = 0;

--bun:split

SELECT 1

--bun:split

SELECT 2
`

const transactionalSQLTemplate = `SET statement_timeout = 0;

SELECT 1;
`

//------------------------------------------------------------------------------

// MigrationSlice is a slice of migrations that provides helper methods for filtering and grouping.
type MigrationSlice []Migration

func (ms MigrationSlice) String() string {
	if len(ms) == 0 {
		return "empty"
	}

	if len(ms) > 5 {
		return fmt.Sprintf("%d migrations (%s ... %s)", len(ms), ms[0].Name, ms[len(ms)-1].Name)
	}

	var sb strings.Builder

	for i := range ms {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(ms[i].String())
	}

	return sb.String()
}

// Applied returns applied migrations in descending order
// (the order is important and is used in Rollback).
func (ms MigrationSlice) Applied() MigrationSlice {
	var applied MigrationSlice
	for i := range ms {
		if ms[i].IsApplied() {
			applied = append(applied, ms[i])
		}
	}
	sortDesc(applied)
	return applied
}

// Unapplied returns unapplied migrations in ascending order
// (the order is important and is used in Migrate).
func (ms MigrationSlice) Unapplied() MigrationSlice {
	var unapplied MigrationSlice
	for i := range ms {
		if !ms[i].IsApplied() {
			unapplied = append(unapplied, ms[i])
		}
	}
	sortAsc(unapplied)
	return unapplied
}

// LastGroupID returns the last applied migration group id.
// The id is 0 when there are no migration groups.
func (ms MigrationSlice) LastGroupID() int64 {
	var lastGroupID int64
	for i := range ms {
		groupID := ms[i].GroupID
		if groupID > lastGroupID {
			lastGroupID = groupID
		}
	}
	return lastGroupID
}

// LastGroup returns the last applied migration group.
func (ms MigrationSlice) LastGroup() *MigrationGroup {
	group := &MigrationGroup{
		ID: ms.LastGroupID(),
	}
	if group.ID == 0 {
		return group
	}
	for i := range ms {
		if ms[i].GroupID == group.ID {
			group.Migrations = append(group.Migrations, ms[i])
		}
	}
	return group
}

func (ms MigrationSlice) Index(migrationName string) int {
	for i := range ms {
		if ms[i].Name == migrationName {
			return i
		}
	}
	return -1
}

// MigrationGroup is a group of migrations that were applied together in a single Migrate call.
type MigrationGroup struct {
	ID         int64
	Migrations MigrationSlice
}

// IsZero reports whether the group is empty.
func (g MigrationGroup) IsZero() bool {
	return g.ID == 0 && len(g.Migrations) == 0
}

func (g MigrationGroup) String() string {
	if g.IsZero() {
		return "nil"
	}
	return fmt.Sprintf("group #%d (%s)", g.ID, g.Migrations)
}

// MigrationFile represents a generated migration file on disk.
type MigrationFile struct {
	Name    string
	Path    string
	Content string
}

//------------------------------------------------------------------------------

type migrationConfig struct {
	nop          bool
	templateData any
}

func (m *Migrator) newMigrationConfig(opts []MigrationOption) *migrationConfig {
	cfg := &migrationConfig{
		templateData: m.templateData,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// MigrationOption configures how a migration is executed.
type MigrationOption func(cfg *migrationConfig)

// WithNopMigration creates a no-op migration that marks itself as applied without running.
func WithNopMigration() MigrationOption {
	return func(cfg *migrationConfig) {
		cfg.nop = true
	}
}

// WithSQLTemplateData provides data for templated SQL migrations.
func WithSQLTemplateData(templateData any) MigrationOption {
	return func(cfg *migrationConfig) {
		cfg.templateData = templateData
	}
}

//------------------------------------------------------------------------------

func sortAsc(ms MigrationSlice) {
	slices.SortFunc(ms, func(a, b Migration) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortDesc(ms MigrationSlice) {
	slices.SortFunc(ms, func(a, b Migration) int {
		return strings.Compare(b.Name, a.Name)
	})
}
