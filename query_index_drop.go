package bun

import (
	"github.com/uptrace/bun/sqlfmt"
)

type DropIndexQuery struct {
	baseQuery
	cascadeQuery

	concurrently bool
	ifExists     bool

	index sqlfmt.QueryWithArgs
}

func NewDropIndexQuery(db *DB) *DropIndexQuery {
	q := &DropIndexQuery{
		baseQuery: baseQuery{
			db:  db,
			dbi: db.DB,
		},
	}
	return q
}

func (q *DropIndexQuery) DB(db DBI) *DropIndexQuery {
	q.dbi = db
	return q
}

func (q *DropIndexQuery) Model(model interface{}) *DropIndexQuery {
	q.setTableModel(model)
	return q
}

//------------------------------------------------------------------------------

func (q *DropIndexQuery) Concurrently() *DropIndexQuery {
	q.concurrently = true
	return q
}

func (q *DropIndexQuery) IfExists() *DropIndexQuery {
	q.ifExists = true
	return q
}

func (q *DropIndexQuery) Restrict() *DropIndexQuery {
	q.restrict = true
	return q
}

func (q *DropIndexQuery) Index(query string, args ...interface{}) *DropIndexQuery {
	q.index = sqlfmt.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *DropIndexQuery) AppendQuery(fmter sqlfmt.QueryFormatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = append(b, "DROP INDEX "...)

	if q.concurrently {
		b = append(b, "CONCURRENTLY "...)
	}
	if q.ifExists {
		b = append(b, "IF EXISTS "...)
	}

	b, err = q.index.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	b = q.appendCascade(fmter, b)

	return b, nil
}
