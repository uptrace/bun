package migrate

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type Migration struct {
	ID         int64 `bun:",autoincrement"`
	Name       string
	GroupID    int64
	MigratedAt time.Time `bun:"default:current_timestamp"`

	Up   MigrationFunc `bun:"-"`
	Down MigrationFunc `bun:"-"`
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

		var dbi bun.DBI

		if isTx {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			dbi = tx
		} else {
			conn, err := db.Conn(ctx)
			if err != nil {
				return err
			}
			dbi = conn
		}

		for _, q := range queries {
			_, err = dbi.ExecContext(ctx, q)
			if err != nil {
				return err
			}
		}

		if tx, ok := dbi.(*sql.Tx); ok {
			return tx.Commit()
		}
		if conn, ok := dbi.(*sql.Conn); ok {
			return conn.Close()
		}
		return nil
	}
}

const goTemplate = `package main
`

const sqlTemplate = `SELECT 1

--bun:split

SELECT 2
`
