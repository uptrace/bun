package bun

import (
	"context"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/sqlfmt"
)

type DropTableQuery struct {
	baseQuery

	ifExists bool
	cascade  bool
}

func NewDropTableQuery(db *DB) *DropTableQuery {
	q := &DropTableQuery{
		baseQuery: baseQuery{
			db:  db,
			dbi: db.DB,
		},
	}
	return q
}

func (q *DropTableQuery) DB(db DBI) *DropTableQuery {
	q.dbi = db
	return q
}

func (q *DropTableQuery) Model(model interface{}) *DropTableQuery {
	q.setTableModel(model)
	return q
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) IfExists() *DropTableQuery {
	q.ifExists = true
	return q
}

func (q *DropTableQuery) Cascade() *DropTableQuery {
	q.cascade = true
	return q
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) AppendQuery(fmter sqlfmt.QueryFormatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}
	if q.table == nil {
		return nil, errModelNil
	}

	b = append(b, "DROP TABLE "...)
	if q.ifExists {
		b = append(b, "IF EXISTS "...)
	}

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.cascade && q.db.features.Has(feature.DropTableCascade) {
		b = append(b, " CASCADE"...)
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *DropTableQuery) Exec(ctx context.Context, dest ...interface{}) (res Result, _ error) {
	queryBytes, err := q.AppendQuery(q.db.fmter, nil)
	if err != nil {
		return res, err
	}
	query := internal.String(queryBytes)

	return q.exec(ctx, q, query)
}
