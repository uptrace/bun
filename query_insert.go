package bun

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type InsertQuery struct {
	whereBaseQuery
	returningQuery
	customValueQuery

	onConflict schema.QueryWithArgs
	setQuery

	ignore  bool
	replace bool
}

func NewInsertQuery(db *DB) *InsertQuery {
	q := &InsertQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db:  db,
				dbi: db.DB,
			},
		},
	}
	return q
}

func (q *InsertQuery) DB(db DBI) *InsertQuery {
	q.setDBI(db)
	return q
}

func (q *InsertQuery) Model(model interface{}) *InsertQuery {
	q.setTableModel(model)
	return q
}

// Apply calls the fn passing the SelectQuery as an argument.
func (q *InsertQuery) Apply(fn func(*InsertQuery) *InsertQuery) *InsertQuery {
	return fn(q)
}

func (q *InsertQuery) With(name string, query schema.QueryAppender) *InsertQuery {
	q.addWith(name, query)
	return q
}

//------------------------------------------------------------------------------

func (q *InsertQuery) Table(tables ...string) *InsertQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *InsertQuery) TableExpr(query string, args ...interface{}) *InsertQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *InsertQuery) ModelTableExpr(query string, args ...interface{}) *InsertQuery {
	q.modelTable = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *InsertQuery) Column(columns ...string) *InsertQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

// Value overwrites model value for the column in INSERT and UPDATE queries.
func (q *InsertQuery) Value(column string, value string, args ...interface{}) *InsertQuery {
	if q.table == nil {
		q.err = errModelNil
		return q
	}
	q.addValue(q.table, column, value, args)
	return q
}

func (q *InsertQuery) Where(query string, args ...interface{}) *InsertQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *InsertQuery) WhereOr(query string, args ...interface{}) *InsertQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " OR "))
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
func (q *InsertQuery) Returning(query string, args ...interface{}) *InsertQuery {
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

func (q *InsertQuery) hasReturning() bool {
	if !q.db.features.Has(feature.Returning) {
		return false
	}
	return q.returningQuery.hasReturning()
}

//------------------------------------------------------------------------------

// Ignore generates an `INSERT IGNORE INTO` query (MySQL).
func (q *InsertQuery) Ignore() *InsertQuery {
	q.ignore = true
	return q
}

// Replaces generates a `REPLACE INTO` query (MySQL).
func (q *InsertQuery) Replace() *InsertQuery {
	q.replace = true
	return q
}

//------------------------------------------------------------------------------

func (q *InsertQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b, err = q.appendWith(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.replace {
		b = append(b, "REPLACE "...)
	} else {
		b = append(b, "INSERT "...)
		if q.ignore {
			b = append(b, "IGNORE "...)
		}
	}
	b = append(b, "INTO "...)

	if !q.onConflict.IsZero() {
		b, err = q.appendFirstTableWithAlias(fmter, b)
	} else {
		b, err = q.appendFirstTable(fmter, b)
	}
	if err != nil {
		return nil, err
	}

	b, err = q.appendColumnsValues(fmter, b)
	if err != nil {
		return nil, err
	}

	b, err = q.appendOn(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.hasReturning() {
		b, err = q.appendReturning(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) appendColumnsValues(
	fmter schema.Formatter, b []byte,
) (_ []byte, err error) {
	if q.hasMultiTables() {
		if q.columns != nil {
			b = append(b, " ("...)
			b, err = q.appendColumns(fmter, b)
			if err != nil {
				return nil, err
			}
			b = append(b, ")"...)
		}

		b = append(b, " SELECT * FROM "...)
		b, err = q.appendOtherTables(fmter, b)
		if err != nil {
			return nil, err
		}

		return b, nil
	}

	if m, ok := q.model.(*mapModel); ok {
		return m.appendColumnsValues(fmter, b), nil
	}
	if _, ok := q.model.(*mapSliceModel); ok {
		return nil, fmt.Errorf("Insert(*[]map[string]interface{}) is not supported")
	}

	if q.model == nil {
		return nil, errModelNil
	}

	fields, err := q.getFields()
	if err != nil {
		return nil, err
	}

	b = append(b, " ("...)
	b = q.appendFields(fmter, b, fields)
	b = append(b, ") VALUES ("...)

	switch model := q.tableModel.(type) {
	case *structTableModel:
		b, err = q.appendStructValues(fmter, b, fields, model.strct)
		if err != nil {
			return nil, err
		}
	case *sliceTableModel:
		b, err = q.appendSliceValues(fmter, b, fields, model.slice)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("bun: Insert does not support %T", q.tableModel)
	}

	b = append(b, ')')

	return b, nil
}

func (q *InsertQuery) appendStructValues(
	fmter schema.Formatter, b []byte, fields []*schema.Field, strct reflect.Value,
) (_ []byte, err error) {
	isTemplate := fmter.IsNop()
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}

		app, ok := q.modelValues[f.Name]
		if ok {
			b, err = app.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
			q.addReturningField(f)
			continue
		}

		switch {
		case isTemplate:
			b = append(b, '?')
		case f.SQLDefault != "" && f.HasZeroValue(strct):
			if q.db.features.Has(feature.DefaultPlaceholder) {
				b = append(b, "DEFAULT"...)
			} else {
				b = append(b, f.SQLDefault...)
			}
			q.addReturningField(f)
		case f.NullZero && f.HasZeroValue(strct):
			if q.db.features.Has(feature.DefaultPlaceholder) {
				b = append(b, "DEFAULT"...)
			} else {
				b = append(b, "NULL"...)
			}
			q.addReturningField(f)
		default:
			b = f.AppendValue(fmter, b, strct)
		}
	}

	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}

		b, err = v.value.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) appendSliceValues(
	fmter schema.Formatter, b []byte, fields []*schema.Field, slice reflect.Value,
) (_ []byte, err error) {
	if fmter.IsNop() {
		return q.appendStructValues(fmter, b, fields, reflect.Value{})
	}

	sliceLen := slice.Len()
	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			b = append(b, "), ("...)
		}
		el := indirect(slice.Index(i))
		b, err = q.appendStructValues(fmter, b, fields, el)
		if err != nil {
			return nil, err
		}
	}

	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}

		b, err = v.value.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) getFields() ([]*schema.Field, error) {
	if q.db.features.Has(feature.DefaultPlaceholder) || len(q.columns) > 0 {
		return q.baseQuery.getFields()
	}

	var strct reflect.Value

	switch model := q.tableModel.(type) {
	case *structTableModel:
		strct = model.strct
	case *sliceTableModel:
		if model.sliceLen == 0 {
			return nil, fmt.Errorf("bun: Insert(empty %T)", model.slice.Type())
		}
		strct = indirect(model.slice.Index(0))
	}

	fields := make([]*schema.Field, 0, len(q.table.Fields))

	for _, f := range q.table.Fields {
		if f.NotNull && f.NullZero && f.SQLDefault == "" && f.HasZeroValue(strct) {
			q.addReturningField(f)
			continue
		}
		fields = append(fields, f)
	}

	return fields, nil
}

func (q *InsertQuery) appendFields(
	fmter schema.Formatter, b []byte, fields []*schema.Field,
) []byte {
	b = appendColumns(b, "", fields)
	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}
		b = fmter.AppendIdent(b, v.column)
	}
	return b
}

//------------------------------------------------------------------------------

func (q *InsertQuery) On(s string, args ...interface{}) *InsertQuery {
	q.onConflict = schema.SafeQuery(s, args)
	return q
}

func (q *InsertQuery) Set(query string, args ...interface{}) *InsertQuery {
	q.addSet(schema.SafeQuery(query, args))
	return q
}

func (q *InsertQuery) appendOn(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.onConflict.IsZero() {
		return b, nil
	}

	b = append(b, " ON "...)
	b, err = q.onConflict.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	if len(q.set) > 0 {
		if fmter.HasFeature(feature.OnDuplicateKey) {
			b = append(b, ' ')
		} else {
			b = append(b, " SET "...)
		}

		b, err = q.appendSet(fmter, b)
		if err != nil {
			return nil, err
		}
	} else if len(q.columns) > 0 {
		fields, err := q.getDataFields()
		if err != nil {
			return nil, err
		}

		if len(fields) == 0 {
			fields = q.tableModel.Table().DataFields
		}

		b = q.appendSetExcluded(b, fields)
	}

	b, err = q.appendWhere(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (q *InsertQuery) appendSetExcluded(b []byte, fields []*schema.Field) []byte {
	b = append(b, " SET "...)
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, f.SQLName...)
		b = append(b, " = EXCLUDED."...)
		b = append(b, f.SQLName...)
	}
	return b
}

//------------------------------------------------------------------------------

func (q *InsertQuery) Exec(ctx context.Context, dest ...interface{}) (res sql.Result, err error) {
	if err := q.beforeInsertQueryHook(ctx); err != nil {
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

	if len(dest) > 0 || q.hasReturning() {
		res, err = q.scan(ctx, q, query, dest, false)
		if err != nil {
			return res, err
		}
	} else {
		res, err = q.exec(ctx, q, query)
		if err != nil {
			return res, err
		}

		if err := q.tryLastInsertID(res, dest); err != nil {
			return res, err
		}
	}

	if err := q.afterInsertQueryHook(ctx); err != nil {
		return res, err
	}

	return res, nil
}

func (q *InsertQuery) beforeInsertQueryHook(ctx context.Context) error {
	if q.tableModel == nil {
		return nil
	}

	if err := q.tableModel.BeforeInsert(ctx); err != nil {
		return err
	}

	// if hook, ok := q.table.ZeroIface.(BeforeInsertQueryHook); ok {
	// 	if err := hook.BeforeInsertQuery(ctx, q); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (q *InsertQuery) afterInsertQueryHook(ctx context.Context) error {
	if q.tableModel == nil {
		return nil
	}

	if err := q.tableModel.AfterInsert(ctx); err != nil {
		return err
	}

	// if hook, ok := q.table.ZeroIface.(AfterInsertQueryHook); ok {
	// 	if err := hook.AfterInsertQuery(ctx, q); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (q *InsertQuery) tryLastInsertID(res sql.Result, dest []interface{}) error {
	if q.db.features.Has(feature.Returning) || q.table == nil || len(q.table.PKs) != 1 {
		return nil
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if id == 0 {
		return nil
	}

	model, err := q.getModel(dest)
	if err != nil {
		return err
	}

	pk := q.table.PKs[0]
	switch model := model.(type) {
	case *structTableModel:
		if err := pk.ScanValue(model.strct, id); err != nil {
			return err
		}
	case *sliceTableModel:
		sliceLen := model.slice.Len()
		for i := 0; i < sliceLen; i++ {
			strct := indirect(model.slice.Index(i))
			if err := pk.ScanValue(strct, id); err != nil {
				return err
			}
			id++
		}
	}

	return nil
}
