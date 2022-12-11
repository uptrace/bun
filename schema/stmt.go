package schema

import (
	"context"
	"database/sql"
)

type Stmt interface {
	Close() error
	Exec(...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, args ...interface{}) (sql.Result, error)
	Query(...interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error)
	QueryRow(args ...interface{}) *sql.Row
	QueryRowContext(ctx context.Context, args ...interface{}) *sql.Row
}
