package bun

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type DropColumnQuery struct {
	baseQuery

	comment string
}

var _ Query = (*DropColumnQuery)(nil)

func NewDropColumnQuery(db *DB) *DropColumnQuery {
	q := &DropColumnQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	return q
}

func (q *DropColumnQuery) Conn(db IConn) *DropColumnQuery {
	q.setConn(db)
	return q
}

func (q *DropColumnQuery) Model(model interface{}) *DropColumnQuery {
	q.setModel(model)
	return q
}

func (q *DropColumnQuery) Err(err error) *DropColumnQuery {
	q.setErr(err)
	return q
}

// Apply calls each function in fns, passing the DropColumnQuery as an argument.
func (q *DropColumnQuery) Apply(fns ...func(*DropColumnQuery) *DropColumnQuery) *DropColumnQuery {
	for _, fn := range fns {
		if fn != nil {
			q = fn(q)
		}
	}
	return q
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Table(tables ...string) *DropColumnQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *DropColumnQuery) TableExpr(query string, args ...interface{}) *DropColumnQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *DropColumnQuery) ModelTableExpr(query string, args ...interface{}) *DropColumnQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Column(columns ...string) *DropColumnQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

func (q *DropColumnQuery) ColumnExpr(query string, args ...interface{}) *DropColumnQuery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *DropColumnQuery) Comment(comment string) *DropColumnQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Operation() string {
	return "DROP COLUMN"
}

func (q *DropColumnQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	if len(q.columns) != 1 {
		return nil, fmt.Errorf("bun: DropColumnQuery requires exactly one column")
	}

	b = append(b, "ALTER TABLE "...)

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " DROP COLUMN "...)

	b, err = q.columns[0].AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {
	// if a comment is propagated via the context, use it
	setCommentFromContext(ctx, q)

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
