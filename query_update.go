package bun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
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
				db:   db,
				conn: db.DB,
			},
		},
	}
	return q
}

func (q *UpdateQuery) Conn(db IConn) *UpdateQuery {
	q.setConn(db)
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

func (q *UpdateQuery) With(name string, query schema.QueryAppender) *UpdateQuery {
	q.addWith(name, query)
	return q
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Table(tables ...string) *UpdateQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *UpdateQuery) TableExpr(query string, args ...interface{}) *UpdateQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *UpdateQuery) ModelTableExpr(query string, args ...interface{}) *UpdateQuery {
	q.modelTable = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Column(columns ...string) *UpdateQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

func (q *UpdateQuery) ExcludeColumn(columns ...string) *UpdateQuery {
	q.excludeColumn(columns)
	return q
}

func (q *UpdateQuery) Set(query string, args ...interface{}) *UpdateQuery {
	q.addSet(schema.SafeQuery(query, args))
	return q
}

// Value overwrites model value for the column in INSERT and UPDATE queries.
func (q *UpdateQuery) Value(column string, value string, args ...interface{}) *UpdateQuery {
	if q.table == nil {
		q.err = errNilModel
		return q
	}
	q.addValue(q.table, column, value, args)
	return q
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) WherePK() *UpdateQuery {
	q.flags = q.flags.Set(wherePKFlag)
	return q
}

func (q *UpdateQuery) Where(query string, args ...interface{}) *UpdateQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *UpdateQuery) WhereOr(query string, args ...interface{}) *UpdateQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " OR "))
	return q
}

func (q *UpdateQuery) WhereGroup(sep string, fn func(*WhereQuery)) *UpdateQuery {
	q.addWhereGroup(sep, fn)
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
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

func (q *UpdateQuery) hasReturning() bool {
	if !q.db.features.Has(feature.Returning) {
		return false
	}
	return q.returningQuery.hasReturning()
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	withAlias := fmter.HasFeature(feature.UpdateMultiTable)

	b, err = q.appendWith(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "UPDATE "...)

	if withAlias {
		b, err = q.appendTablesWithAlias(fmter, b)
	} else {
		b, err = q.appendFirstTableWithAlias(fmter, b)
	}
	if err != nil {
		return nil, err
	}

	b, err = q.mustAppendSet(fmter, b)
	if err != nil {
		return nil, err
	}

	if !fmter.HasFeature(feature.UpdateMultiTable) {
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

func (q *UpdateQuery) mustAppendSet(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, " SET "...)

	if len(q.set) > 0 {
		return q.appendSet(fmter, b)
	}

	if m, ok := q.model.(*mapModel); ok {
		return m.appendSet(fmter, b), nil
	}

	if q.tableModel == nil {
		return nil, errNilModel
	}

	switch model := q.tableModel.(type) {
	case *structTableModel:
		b, err = q.appendSetStruct(fmter, b, model)
		if err != nil {
			return nil, err
		}
	case *sliceTableModel:
		return nil, errors.New("bun: to bulk Update, use CTE and VALUES")
	default:
		return nil, fmt.Errorf("bun: Update does not support %T", q.tableModel)
	}

	return b, nil
}

func (q *UpdateQuery) appendSetStruct(
	fmter schema.Formatter, b []byte, model *structTableModel,
) ([]byte, error) {
	fields, err := q.getDataFields()
	if err != nil {
		return nil, err
	}

	isTemplate := fmter.IsNop()
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

func (q *UpdateQuery) appendOtherTables(fmter schema.Formatter, b []byte) (_ []byte, err error) {
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

func (q *UpdateQuery) Bulk() *UpdateQuery {
	model, ok := q.model.(*sliceTableModel)
	if !ok {
		q.setErr(fmt.Errorf("bun: Bulk requires a slice, got %T", q.model))
		return q
	}

	return q.With("_data", q.db.NewValues(model)).
		Model(model).
		TableExpr("_data").
		Set(q.updateSliceSet(model)).
		Where(q.updateSliceWhere(model))
}

func (q *UpdateQuery) updateSliceSet(model *sliceTableModel) string {
	var b []byte
	for i, field := range model.table.DataFields {
		if i > 0 {
			b = append(b, ", "...)
		}
		if q.db.fmter.HasFeature(feature.UpdateMultiTable) {
			b = append(b, model.table.SQLAlias...)
			b = append(b, '.')
		}
		b = append(b, field.SQLName...)
		b = append(b, " = _data."...)
		b = append(b, field.SQLName...)
	}
	return internal.String(b)
}

func (db *UpdateQuery) updateSliceWhere(model *sliceTableModel) string {
	var b []byte
	for i, pk := range model.table.PKs {
		if i > 0 {
			b = append(b, " AND "...)
		}
		b = append(b, model.table.SQLAlias...)
		b = append(b, '.')
		b = append(b, pk.SQLName...)
		b = append(b, " = _data."...)
		b = append(b, pk.SQLName...)
	}
	return internal.String(b)
}

//------------------------------------------------------------------------------

func (q *UpdateQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {
	if q.table != nil {
		if err := q.beforeUpdateHook(ctx); err != nil {
			return nil, err
		}
	}

	bs := getByteSlice()
	defer putByteSlice(bs)

	queryBytes, err := q.AppendQuery(q.db.fmter, bs.b)
	if err != nil {
		return nil, err
	}

	bs.update(queryBytes)
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
		if err := q.afterUpdateHook(ctx); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (q *UpdateQuery) beforeUpdateHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(BeforeUpdateHook); ok {
		if err := hook.BeforeUpdate(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (q *UpdateQuery) afterUpdateHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(AfterUpdateHook); ok {
		if err := hook.AfterUpdate(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// FQN returns a fully qualified column name. For MySQL, it returns the column name with
// the table alias. For other RDBMS, it returns just the column name.
func (q *UpdateQuery) FQN(name string) Ident {
	if q.db.fmter.HasFeature(feature.UpdateMultiTable) {
		return Ident(q.table.Alias + "." + name)
	}
	return Ident(name)
}
