package bun

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type AddColumnQuery struct {
	baseQuery

	ifNotExists bool
	comment     string
}

var _ Query = (*AddColumnQuery)(nil)

func NewAddColumnQuery(db *DB) *AddColumnQuery {
	q := &AddColumnQuery{
		baseQuery: baseQuery{
			db: db,
		},
	}
	return q
}

func (q *AddColumnQuery) Conn(db IConn) *AddColumnQuery {
	q.setConn(db)
	return q
}

func (q *AddColumnQuery) Model(model interface{}) *AddColumnQuery {
	q.setModel(model)
	return q
}

func (q *AddColumnQuery) Err(err error) *AddColumnQuery {
	q.setErr(err)
	return q
}

// Apply calls each function in fns, passing the AddColumnQuery as an argument.
func (q *AddColumnQuery) Apply(fns ...func(*AddColumnQuery) *AddColumnQuery) *AddColumnQuery {
	for _, fn := range fns {
		if fn != nil {
			q = fn(q)
		}
	}
	return q
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) Table(tables ...string) *AddColumnQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *AddColumnQuery) TableExpr(query string, args ...interface{}) *AddColumnQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *AddColumnQuery) ModelTableExpr(query string, args ...interface{}) *AddColumnQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) ColumnExpr(query string, args ...interface{}) *AddColumnQuery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

func (q *AddColumnQuery) IfNotExists() *AddColumnQuery {
	q.ifNotExists = true
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *AddColumnQuery) Comment(comment string) *AddColumnQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) Operation() string {
	return "ADD COLUMN"
}

func (q *AddColumnQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	if len(q.columns) != 1 {
		return nil, fmt.Errorf("bun: AddColumnQuery requires exactly one column")
	}

	b = append(b, "ALTER TABLE "...)

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " ADD "...)

	if q.ifNotExists {
		b = append(b, "IF NOT EXISTS "...)
	}

	b, err = q.columns[0].AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *AddColumnQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {
	if q.ifNotExists && !q.hasFeature(feature.AlterColumnExists) {
		return nil, feature.NewNotSupportError(feature.AlterColumnExists)
	}

	queryBytes, err := q.AppendQuery(q.db.fmter, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	query := internal.String(queryBytes)
	return q.exec(ctx, q, query)
}
