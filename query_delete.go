package bun

import (
	"context"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/sqlfmt"
)

type DeleteQuery struct {
	whereBaseQuery
	returningQuery
}

func NewDeleteQuery(db *DB) *DeleteQuery {
	q := &DeleteQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db:  db,
				dbi: db.DB,
			},
		},
	}
	return q
}

func (q *DeleteQuery) DB(db DBI) *DeleteQuery {
	q.dbi = db
	return q
}

func (q *DeleteQuery) Model(model interface{}) *DeleteQuery {
	q.setTableModel(model)
	return q
}

// Apply calls the fn passing the DeleteQuery as an argument.
func (q *DeleteQuery) Apply(fn func(*DeleteQuery) *DeleteQuery) *DeleteQuery {
	return fn(q)
}

func (q *DeleteQuery) With(name string, query sqlfmt.QueryAppender) *DeleteQuery {
	q.addWith(name, query)
	return q
}

func (q *DeleteQuery) Table(tables ...string) *DeleteQuery {
	for _, table := range tables {
		q.addTable(sqlfmt.UnsafeIdent(table))
	}
	return q
}

func (q *DeleteQuery) TableExpr(query string, args ...interface{}) *DeleteQuery {
	q.addTable(sqlfmt.SafeQuery(query, args))
	return q
}

func (q *DeleteQuery) ModelTableExpr(query string, args ...interface{}) *DeleteQuery {
	q.modelTable = sqlfmt.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *DeleteQuery) Where(query string, args ...interface{}) *DeleteQuery {
	q.addWhere(sqlfmt.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *DeleteQuery) WhereOr(query string, args ...interface{}) *DeleteQuery {
	q.addWhere(sqlfmt.SafeQueryWithSep(query, args, " OR "))
	return q
}

func (q *DeleteQuery) WhereGroup(sep string, fn func(*WhereQuery)) *DeleteQuery {
	q.addWhereGroup(sep, fn)
	return q
}

// WherePK adds conditions based on the model primary keys.
// Usually it is the same as:
//
//    Where("id = ?id")
func (q *DeleteQuery) WherePK() *DeleteQuery {
	q.flags = q.flags.Set(wherePKFlag)
	return q
}

func (q *DeleteQuery) WhereDeleted() *DeleteQuery {
	q.whereDeleted()
	return q
}

func (q *DeleteQuery) WhereAllWithDeleted() *DeleteQuery {
	q.whereAllWithDeleted()
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
func (q *DeleteQuery) Returning(query string, args ...interface{}) *DeleteQuery {
	q.addReturning(sqlfmt.SafeQuery(query, args))
	return q
}

func (q *DeleteQuery) hasReturning() bool {
	if !q.db.features.Has(feature.Returning) {
		return false
	}
	return q.returningQuery.hasReturning()
}

//------------------------------------------------------------------------------

func (q *DeleteQuery) AppendQuery(fmter sqlfmt.QueryFormatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b, err = q.appendWith(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "DELETE FROM "...)
	b, err = q.appendFirstTableWithAlias(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.hasMultiTables() {
		b = append(b, " USING "...)
		b, err = q.appendOtherTables(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	b, err = q.mustAppendWhere(fmter, b)
	if err != nil {
		return nil, err
	}

	if len(q.returning) > 0 {
		b, err = q.appendReturning(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *DeleteQuery) Exec(ctx context.Context, dest ...interface{}) (res Result, _ error) {
	if q.tableModel == nil || q.table.SoftDeleteField == nil {
		return q.ForceDelete(ctx, dest...)
	}

	if err := q.tableModel.updateSoftDeleteField(); err != nil {
		return res, err
	}

	upd := &UpdateQuery{
		whereBaseQuery: q.whereBaseQuery,
	}
	upd = upd.Column(q.table.SoftDeleteField.Name)

	return upd.Exec(ctx, dest...)
}

func (q *DeleteQuery) ForceDelete(ctx context.Context, dest ...interface{}) (res Result, err error) {
	if q.table != nil && q.table.SoftDeleteField != nil {
		q = q.WhereAllWithDeleted()
	}

	if q.tableModel != nil {
		ctx, err = q.tableModel.BeforeDelete(ctx)
		if err != nil {
			return res, err
		}
	}

	queryBytes, err := q.AppendQuery(q.db.fmter, nil)
	if err != nil {
		return res, err
	}
	query := internal.String(queryBytes)

	if q.hasReturning() {
		res, err = q.scan(ctx, q, query, dest)
	} else {
		res, err = q.exec(ctx, q, query)
	}
	if err != nil {
		return res, err
	}

	if q.tableModel != nil {
		if err := q.tableModel.AfterDelete(ctx); err != nil {
			return res, err
		}
	}

	return res, nil
}
