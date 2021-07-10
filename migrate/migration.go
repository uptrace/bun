package migrate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type Migration struct {
	bun.BaseModel

	ID         int64
	Name       string
	GroupID    int64
	MigratedAt time.Time `bun:",notnull,nullzero,default:current_timestamp"`

	Up   MigrationFunc `bun:"-"`
	Down MigrationFunc `bun:"-"`
}

func (m *Migration) String() string {
	return m.Name
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

		for _, q := range queries {
			_, err = idb.ExecContext(ctx, q)
			if err != nil {
				return err
			}
		}

		if tx, ok := idb.(bun.Tx); ok {
			return tx.Commit()
		} else if conn, ok := idb.(bun.Conn); ok {
			return conn.Close()
		}

		panic("not reached")
	}
}

const goTemplate = `package main
`

const sqlTemplate = `SELECT 1

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

type MigrationGroup struct {
	ID         int64
	Migrations MigrationSlice
}

func (g *MigrationGroup) String() string {
	if g.ID == 0 && len(g.Migrations) == 0 {
		return "nil"
	}
	return fmt.Sprintf("group #%d (%s)", g.ID, g.Migrations)
}

type MigrationFile struct {
	FileName string
	FilePath string
	Content  string
}

//------------------------------------------------------------------------------

type migrationConfig struct {
	dryRun bool
}

func newMigrationConfig(opts []MigrationOption) *migrationConfig {
	cfg := new(migrationConfig)
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

type MigrationOption func(cfg *migrationConfig)

func WithMigrationDryRun() MigrationOption {
	return func(cfg *migrationConfig) {
		cfg.dryRun = true
	}
}
