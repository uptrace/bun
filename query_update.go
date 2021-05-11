package bun

import (
	"context"
	"fmt"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/sqlfmt"
)

type UpdateQuery struct {
	whereBaseQuery
	returningQuery
	customValueQuery
	setQuery

	omitZero bool
}

func NewUpdateQuery(db *DB) *UpdateQuery {
	q := &UpdateQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db:  db,
				dbi: db.DB,
			},
		},
	}
	return q
}

func (q *UpdateQuery) DB(db DBI) *UpdateQuery {
	q.dbi = db
	return q
}

func (q *UpdateQuery) Model(model interface{}) *UpdateQuery {
	q.setTableModel(model)
	return q
}

// Apply calls the fn passing the SelectQuery as an argument.
func (q *UpdateQuery) Apply(fn func(*UpdateQuery) *UpdateQuery) *UpdateQuery {
	return fn(q)
}

func (q *UpdateQuery) With(name string, query sqlfmt.QueryAppender) *UpdateQuery {
	q.addWith(name, query)
	return q
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Table(tables ...string) *UpdateQuery {
	for _, table := range tables {
		q.addTable(sqlfmt.UnsafeIdent(table))
	}
	return q
}

func (q *UpdateQuery) TableExpr(query string, args ...interface{}) *UpdateQuery {
	q.addTable(sqlfmt.SafeQuery(query, args))
	return q
}

func (q *UpdateQuery) ModelTableExpr(query string, args ...interface{}) *UpdateQuery {
	q.modelTable = sqlfmt.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Column(columns ...string) *UpdateQuery {
	for _, column := range columns {
		q.addColumn(sqlfmt.UnsafeIdent(column))
	}
	return q
}

func (q *UpdateQuery) Set(query string, args ...interface{}) *UpdateQuery {
	q.addSet(sqlfmt.SafeQuery(query, args))
	return q
}

// Value overwrites model value for the column in INSERT and UPDATE queries.
func (q *UpdateQuery) Value(column string, value string, args ...interface{}) *UpdateQuery {
	if q.table == nil {
		q.err = errModelNil
		return q
	}
	q.addValue(q.table, column, value, args)
	return q
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Where(query string, args ...interface{}) *UpdateQuery {
	q.addWhere(sqlfmt.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *UpdateQuery) WhereOr(query string, args ...interface{}) *UpdateQuery {
	q.addWhere(sqlfmt.SafeQueryWithSep(query, args, " OR "))
	return q
}

func (q *UpdateQuery) WhereGroup(sep string, fn func(*WhereQuery)) *UpdateQuery {
	q.addWhereGroup(sep, fn)
	return q
}

// WherePK adds conditions based on the model primary keys.
// Usually it is the same as:
//
//    Where("id = ?id")
func (q *UpdateQuery) WherePK() *UpdateQuery {
	q.flags = q.flags.Set(wherePKFlag)
	return q
}

func (q *UpdateQuery) WhereDeleted() *UpdateQuery {
	q.whereDeleted()
	return q
}

func (q *UpdateQuery) WhereAllWithDeleted() *UpdateQuery {
	q.whereAllWithDeleted()
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
func (q *UpdateQuery) Returning(query string, args ...interface{}) *UpdateQuery {
	q.addReturning(sqlfmt.SafeQuery(query, args))
	return q
}

func (q *UpdateQuery) hasReturning() bool {
	if !q.db.features.Has(feature.Returning) {
		return false
	}
	return q.returningQuery.hasReturning()
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) AppendQuery(fmter sqlfmt.QueryFormatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b, err = q.appendWith(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "UPDATE "...)

	b, err = q.appendFirstTableWithAlias(fmter, b)
	if err != nil {
		return nil, err
	}

	b, err = q.mustAppendSet(fmter, b)
	if err != nil {
		return nil, err
	}

	b, err = q.appendOtherTables(fmter, b)
	if err != nil {
		return nil, err
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

func (q *UpdateQuery) mustAppendSet(fmter sqlfmt.QueryFormatter, b []byte) (_ []byte, err error) {
	if len(q.set) > 0 {
		return q.appendSet(fmter, b)
	}

	b = append(b, " SET "...)

	if m, ok := q.model.(*mapModel); ok {
		return m.appendSet(fmter, b), nil
	}

	if q.tableModel == nil {
		return nil, errModelNil
	}

	switch model := q.tableModel.(type) {
	case *structTableModel:
		b, err = q.appendSetStruct(fmter, b, model)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("bun: Update does not support %T", q.tableModel)
	}

	return b, nil
}

func (q *UpdateQuery) appendSetStruct(
	fmter sqlfmt.QueryFormatter, b []byte, model *structTableModel,
) ([]byte, error) {
	fields, err := q.getDataFields()
	if err != nil {
		return nil, err
	}

	isTemplate := sqlfmt.IsNopFormatter(fmter)
	pos := len(b)
	for _, f := range fields {
		if q.omitZero && f.NullZero && f.HasZeroValue(model.strct) {
			continue
		}

		if len(b) != pos {
			b = append(b, ", "...)
			pos = len(b)
		}

		b = append(b, f.SQLName...)
		b = append(b, " = "...)

		if isTemplate {
			b = append(b, '?')
			continue
		}

		app, ok := q.modelValues[f.Name]
		if ok {
			b, err = app.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		} else {
			b = f.AppendValue(fmter, b, model.strct)
		}
	}

	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}

		b = append(b, v.column...)
		b = append(b, " = "...)

		b, err = v.value.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *UpdateQuery) appendOtherTables(
	fmter sqlfmt.QueryFormatter, b []byte,
) (_ []byte, err error) {
	if !q.hasMultiTables() {
		return b, nil
	}

	b = append(b, " FROM "...)

	b, err = q.whereBaseQuery.appendOtherTables(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Exec(ctx context.Context, dest ...interface{}) (res Result, err error) {
	if err := q.beforeUpdateQueryHook(ctx); err != nil {
		return res, err
	}

	bs := getByteSlice()
	defer putByteSlice(bs)

	queryBytes, err := q.AppendQuery(q.db.fmter, bs.b)
	if err != nil {
		return res, err
	}

	bs.b = queryBytes
	query := internal.String(queryBytes)

	if q.hasReturning() {
		res, err = q.scan(ctx, q, query, dest)
	} else {
		res, err = q.exec(ctx, q, query)
	}
	if err != nil {
		return res, err
	}

	if err := q.afterUpdateQueryHook(ctx); err != nil {
		return res, err
	}

	return res, nil
}

func (q *UpdateQuery) beforeUpdateQueryHook(ctx context.Context) error {
	if q.tableModel == nil {
		return nil
	}

	if err := q.tableModel.BeforeUpdate(ctx); err != nil {
		return err
	}

	// if hook, ok := q.table.ZeroIface.(BeforeUpdateQueryHook); ok {
	// 	if err := hook.BeforeUpdateQuery(ctx, q); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (q *UpdateQuery) afterUpdateQueryHook(ctx context.Context) error {
	if q.tableModel == nil {
		return nil
	}

	if err := q.tableModel.AfterUpdate(ctx); err != nil {
		return err
	}

	// if hook, ok := q.table.ZeroIface.(AfterUpdateQueryHook); ok {
	// 	if err := hook.AfterUpdateQuery(ctx, q); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}
