package bun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type MergeQuery struct {
	baseQuery
	returningQuery

	using schema.QueryWithArgs
	on    schema.QueryWithArgs
	when  []schema.QueryWithArgs
}

var _ Query = (*MergeQuery)(nil)

func NewMergeQuery(db *DB) *MergeQuery {
	q := &MergeQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
	}
	return q
}

func (q *MergeQuery) Conn(db IConn) *MergeQuery {
	q.setConn(db)
	return q
}

func (q *MergeQuery) Model(model interface{}) *MergeQuery {
	q.setModel(model)
	return q
}

// Apply calls the fn passing the SelectQuery as an argument.
func (q *MergeQuery) Apply(fn func(*MergeQuery) *MergeQuery) *MergeQuery {
	return fn(q)
}

func (q *MergeQuery) With(name string, query schema.QueryAppender) *MergeQuery {
	q.addWith(name, query, false)
	return q
}

func (q *MergeQuery) WithRecursive(name string, query schema.QueryAppender) *MergeQuery {
	q.addWith(name, query, true)
	return q
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Table(tables ...string) *MergeQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *MergeQuery) TableExpr(query string, args ...interface{}) *MergeQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *MergeQuery) ModelTableExpr(query string, args ...interface{}) *MergeQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Column(columns ...string) *MergeQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

func (q *MergeQuery) ColumnExpr(query string, args ...interface{}) *MergeQuery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

func (q *MergeQuery) ExcludeColumn(columns ...string) *MergeQuery {
	q.excludeColumn(columns)
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
func (q *MergeQuery) Returning(query string, args ...interface{}) *MergeQuery {
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Operation() string {
	return "MERGE"
}

func (q *MergeQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	fmter = formatterWithModel(fmter, q)

	b, err = q.appendWith(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "MERGE "...)

	if q.db.features.Has(feature.InsertTableAlias) && !q.on.IsZero() {
		b, err = q.appendFirstTableWithAlias(fmter, b)
	} else {
		b, err = q.appendFirstTable(fmter, b)
	}
	if err != nil {
		return nil, err
	}

	b = append(b, " USING "...)
	b, err = q.using.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " ON "...)
	b, err = q.on.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	for _, w := range q.when {
		b = append(b, " WHEN "...)
		b, err = w.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	if q.hasFeature(feature.Output) && q.hasReturning() {
		b = append(b, " OUTPUT "...)
		b, err = q.appendOutput(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	//A MERGE statement must be terminated by a semi-colon (;).
	b = append(b, ";"...)

	return b, nil
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Using(s string, args ...interface{}) *MergeQuery {
	q.using = schema.SafeQuery(s, args)
	return q
}

func (q *MergeQuery) On(s string, args ...interface{}) *MergeQuery {
	q.on = schema.SafeQuery(s, args)
	return q
}

func (q *MergeQuery) When(s string, args ...interface{}) *MergeQuery {
	q.when = append(q.when, schema.SafeQuery(s, args))
	return q
}

//------------------------------------------------------------------------------

func (q *MergeQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {

	if q.err != nil {
		return nil, q.err
	}
	if err := q.beforeAppendModel(ctx, q); err != nil {
		return nil, err
	}

	queryBytes, err := q.AppendQuery(q.db.fmter, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	query := internal.String(queryBytes)
	var res sql.Result

	if hasDest := len(dest) > 0; hasDest ||
		(q.hasReturning() && q.hasFeature(feature.InsertReturning|feature.Output)) {
		model, err := q.getModel(dest)
		if err != nil {
			return nil, err
		}

		res, err = q.scan(ctx, q, query, model, hasDest)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = q.exec(ctx, q, query)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (q *MergeQuery) String() string {
	buf, err := q.AppendQuery(q.db.Formatter(), nil)
	if err != nil {
		panic(err)
	}

	return string(buf)
}
