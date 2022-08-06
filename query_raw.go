package bun

import (
	"context"

	"github.com/uptrace/bun/schema"
)

type RawQuery struct {
	baseQuery

	query string
	args  []interface{}
}

// Deprecated: Use NewRaw instead. When add it to IDB, it conflicts with the sql.Conn#Raw
func (db *DB) Raw(query string, args ...interface{}) *RawQuery {
	return &RawQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		query: query,
		args:  args,
	}
}

func NewRawQuery(db *DB, query string, args ...interface{}) *RawQuery {
	return &RawQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		query: query,
		args:  args,
	}
}

func (q *RawQuery) Conn(db IConn) *RawQuery {
	q.setConn(db)
	return q
}

func (q *RawQuery) Scan(ctx context.Context, dest ...interface{}) error {
	if q.err != nil {
		return q.err
	}

	model, err := q.getModel(dest)
	if err != nil {
		return err
	}

	query := q.db.format(q.query, q.args)
	_, err = q.scan(ctx, q, query, model, true)
	return err
}

func (q *RawQuery) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return fmter.AppendQuery(b, q.query, q.args...), nil
}

func (q *RawQuery) Operation() string {
	return "SELECT"
}
