package bun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type DropIndexQuery struct {
	baseQuery
	cascadeQuery

	concurrently bool
	ifExists     bool

	index   schema.QueryWithArgs
	comment string
}

var _ Query = (*DropIndexQuery)(nil)

func NewDropIndexQuery(db *DB) *DropIndexQuery {
	q := &DropIndexQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	return q
}

func (q *DropIndexQuery) Conn(db IConn) *DropIndexQuery {
	q.setConn(db)
	return q
}

func (q *DropIndexQuery) Model(model interface{}) *DropIndexQuery {
	q.setModel(model)
	return q
}

func (q *DropIndexQuery) Err(err error) *DropIndexQuery {
	q.setErr(err)
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

func (q *DropIndexQuery) Cascade() *DropIndexQuery {
	q.cascade = true
	return q
}

func (q *DropIndexQuery) Restrict() *DropIndexQuery {
	q.restrict = true
	return q
}

func (q *DropIndexQuery) Index(query string, args ...interface{}) *DropIndexQuery {
	q.index = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *DropIndexQuery) Comment(comment string) *DropIndexQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *DropIndexQuery) Operation() string {
	return "DROP INDEX"
}

func (q *DropIndexQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

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

//------------------------------------------------------------------------------

func (q *DropIndexQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {
	queryBytes, err := q.AppendQuery(q.db.fmter, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	query := internal.String(queryBytes)

	res, err := q.exec(ctx, q, query)
	if err != nil {
		return nil, err
	}

	return res, nil
}
