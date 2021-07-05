package bun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type DeleteQuery struct {
	whereBaseQuery
	returningQuery
}

func NewDeleteQuery(db *DB) *DeleteQuery {
	q := &DeleteQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db:   db,
				conn: db.DB,
			},
		},
	}
	return q
}

func (q *DeleteQuery) Conn(db IConn) *DeleteQuery {
	q.setConn(db)
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

func (q *DeleteQuery) With(name string, query schema.QueryAppender) *DeleteQuery {
	q.addWith(name, query)
	return q
}

func (q *DeleteQuery) Table(tables ...string) *DeleteQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *DeleteQuery) TableExpr(query string, args ...interface{}) *DeleteQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *DeleteQuery) ModelTableExpr(query string, args ...interface{}) *DeleteQuery {
	q.modelTable = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *DeleteQuery) WherePK() *DeleteQuery {
	q.flags = q.flags.Set(wherePKFlag)
	return q
}

func (q *DeleteQuery) Where(query string, args ...interface{}) *DeleteQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *DeleteQuery) WhereOr(query string, args ...interface{}) *DeleteQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " OR "))
	return q
}

func (q *DeleteQuery) WhereGroup(sep string, fn func(*WhereQuery)) *DeleteQuery {
	q.addWhereGroup(sep, fn)
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

func (q *DeleteQuery) ForceDelete() *DeleteQuery {
	q.flags = q.flags.Set(forceDeleteFlag)
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
func (q *DeleteQuery) Returning(query string, args ...interface{}) *DeleteQuery {
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

func (q *DeleteQuery) hasReturning() bool {
	if !q.db.features.Has(feature.Returning) {
		return false
	}
	return q.returningQuery.hasReturning()
}

//------------------------------------------------------------------------------

func (q *DeleteQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	if q.isSoftDelete() {
		if err := q.tableModel.updateSoftDeleteField(); err != nil {
			return nil, err
		}

		upd := UpdateQuery{
			whereBaseQuery: q.whereBaseQuery,
			returningQuery: q.returningQuery,
		}
		upd.Column(q.table.SoftDeleteField.Name)
		return upd.AppendQuery(fmter, b)
	}

	q = q.WhereAllWithDeleted()
	withAlias := q.db.features.Has(feature.DeleteTableAlias)

	b, err = q.appendWith(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "DELETE FROM "...)

	if withAlias {
		b, err = q.appendFirstTableWithAlias(fmter, b)
	} else {
		b, err = q.appendFirstTable(fmter, b)
	}
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

	b, err = q.mustAppendWhere(fmter, b, withAlias)
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

func (q *DeleteQuery) isSoftDelete() bool {
	return q.tableModel != nil && q.table.SoftDeleteField != nil && !q.flags.Has(forceDeleteFlag)
}

//------------------------------------------------------------------------------

func (q *DeleteQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {
	if q.table != nil {
		if err := q.beforeDeleteHook(ctx); err != nil {
			return nil, err
		}
	}

	queryBytes, err := q.AppendQuery(q.db.fmter, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	query := internal.String(queryBytes)

	var res sql.Result

	if hasDest := len(dest) > 0; hasDest || q.hasReturning() {
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

	if q.table != nil {
		if err := q.afterDeleteHook(ctx); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (q *DeleteQuery) beforeDeleteHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(BeforeDeleteHook); ok {
		if err := hook.BeforeDelete(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (q *DeleteQuery) afterDeleteHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(AfterDeleteHook); ok {
		if err := hook.AfterDelete(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
