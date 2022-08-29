package migrate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

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

func (m Migration) String() string {
	return fmt.Sprintf("%s_%s", m.Name, m.Comment)
}

func (m Migration) IsApplied() bool {
	return m.ID > 0
}

type MigrationFunc func(ctx context.Context, db *bun.DB) error

func NewSQLMigrationFunc(fsys fs.FS, name string) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		isTx := strings.HasSuffix(name, ".tx.up.sql") || strings.HasSuffix(name, ".tx.down.sql")

		f, err := fsys.Open(name)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(f)
		var queries []string

		var query []byte
		for scanner.Scan() {
			b := scanner.Bytes()

			const prefix = "--bun:"
			if bytes.HasPrefix(b, []byte(prefix)) {
				b = b[len(prefix):]
				if bytes.Equal(b, []byte("split")) {
					queries = append(queries, string(query))
					query = query[:0]
					continue
				}
				return fmt.Errorf("bun: unknown directive: %q", b)
			}

			query = append(query, b...)
			query = append(query, '\n')
		}

		if len(query) > 0 {
			queries = append(queries, string(query))
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		var idb bun.IConn

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

		defer func() {
			if tx, ok := idb.(bun.Tx); ok {
				retErr = tx.Commit()
				return
			}

			if conn, ok := idb.(bun.Conn); ok {
				retErr = conn.Close()
				return
			}

			panic("not reached")
		}()

		for _, q := range queries {
			_, err = idb.ExecContext(ctx, q)
			if err != nil {
				return err
			}
		}

		return retErr
	}
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

//------------------------------------------------------------------------------

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
		sb.WriteString(ms[i].Name)
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

type MigrationGroup struct {
	ID         int64
	Migrations MigrationSlice
}

func (g MigrationGroup) IsZero() bool {
	return g.ID == 0 && len(g.Migrations) == 0
}

func (g MigrationGroup) String() string {
	if g.IsZero() {
		return "nil"
	}
	return fmt.Sprintf("group #%d (%s)", g.ID, g.Migrations)
}

type MigrationFile struct {
	Name    string
	Path    string
	Content string
}

//------------------------------------------------------------------------------

type migrationConfig struct {
	nop bool
}

func newMigrationConfig(opts []MigrationOption) *migrationConfig {
	cfg := new(migrationConfig)
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

type MigrationOption func(cfg *migrationConfig)

func WithNopMigration() MigrationOption {
	return func(cfg *migrationConfig) {
		cfg.nop = true
	}
}

//------------------------------------------------------------------------------

func sortAsc(ms MigrationSlice) {
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Name < ms[j].Name
	})
}

func sortDesc(ms MigrationSlice) {
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Name > ms[j].Name
	})
}
