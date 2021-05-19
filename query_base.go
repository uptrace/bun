package bun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

const (
	wherePKFlag internal.Flag = 1 << iota
	deletedFlag
	allWithDeletedFlag
)

type withQuery struct {
	name  string
	query schema.QueryAppender
}

type DBI interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type baseQuery struct {
	db  *DB
	dbi DBI

	model model
	err   error

	tableModel tableModel
	table      *schema.Table

	with       []withQuery
	modelTable schema.QueryWithArgs
	tables     []schema.QueryWithArgs
	columns    []schema.QueryWithArgs

	flags internal.Flag
}

func (q *baseQuery) GetDB() *DB {
	return q.db
}

func (q *baseQuery) setTableModel(modeli interface{}) {
	model, err := newSingleModel(q.db, modeli)
	if err != nil {
		q.setErr(err)
		return
	}

	q.model = model
	if tm, ok := model.(tableModel); ok {
		q.tableModel = tm
		q.table = tm.Table()
	}
}

func (q *baseQuery) setErr(err error) {
	if q.err == nil {
		q.err = err
	}
}

func (q *baseQuery) getModel(dest []interface{}) (model, error) {
	if len(dest) == 0 {
		return q.model, nil
	}
	return newModel(q.db, dest)
}

//------------------------------------------------------------------------------

func (q *baseQuery) checkSoftDelete() error {
	if q.table == nil {
		return errors.New("bun: can't use soft deletes without a table")
	}
	if q.table.SoftDeleteField == nil {
		return fmt.Errorf("%s does not have a soft delete field", q.table)
	}
	if q.tableModel == nil {
		return errors.New("bun: can't use soft deletes without a table model")
	}
	return nil
}

// Deleted adds `WHERE deleted_at IS NOT NULL` clause for soft deleted models.
func (q *baseQuery) whereDeleted() {
	if err := q.checkSoftDelete(); err != nil {
		q.setErr(err)
		return
	}
	q.flags = q.flags.Set(deletedFlag)
	q.flags = q.flags.Remove(allWithDeletedFlag)
}

// AllWithDeleted changes query to return all rows including soft deleted ones.
func (q *baseQuery) whereAllWithDeleted() {
	if err := q.checkSoftDelete(); err != nil {
		q.setErr(err)
		return
	}
	q.flags = q.flags.Set(allWithDeletedFlag)
	q.flags = q.flags.Remove(deletedFlag)
}

func (q *baseQuery) isSoftDelete() bool {
	if q.table != nil {
		return q.table.SoftDeleteField != nil && !q.flags.Has(allWithDeletedFlag)
	}
	return false
}

//------------------------------------------------------------------------------

func (q *baseQuery) addWith(name string, query schema.QueryAppender) {
	q.with = append(q.with, withQuery{
		name:  name,
		query: query,
	})
}

func (q *baseQuery) appendWith(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if len(q.with) == 0 {
		return b, nil
	}

	b = append(b, "WITH "...)
	for i, with := range q.with {
		if i > 0 {
			b = append(b, ", "...)
		}

		b = fmter.AppendIdent(b, with.name)
		if q, ok := with.query.(schema.ColumnsAppender); ok {
			b = append(b, " ("...)
			b, err = q.AppendColumns(fmter, b)
			if err != nil {
				return nil, err
			}
			b = append(b, ")"...)
		}

		b = append(b, " AS ("...)

		b, err = with.query.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}

		b = append(b, ')')
	}
	b = append(b, ' ')
	return b, nil
}

//------------------------------------------------------------------------------

func (q *baseQuery) addTable(table schema.QueryWithArgs) {
	q.tables = append(q.tables, table)
}

func (q *baseQuery) addColumn(column schema.QueryWithArgs) {
	q.columns = append(q.columns, column)
}

func (q *baseQuery) excludeColumn(columns []string) {
	if q.columns == nil {
		for _, f := range q.table.Fields {
			q.columns = append(q.columns, schema.UnsafeIdent(f.Name))
		}
	}

	if len(columns) == 1 && columns[0] == "*" {
		q.columns = make([]schema.QueryWithArgs, 0)
		return
	}

	for _, column := range columns {
		if !q._excludeColumn(column) {
			q.setErr(fmt.Errorf("bun: can't find column=%q", column))
			return
		}
	}
}

func (q *baseQuery) _excludeColumn(column string) bool {
	for i, col := range q.columns {
		if col.Args == nil && col.Query == column {
			q.columns = append(q.columns[:i], q.columns[i+1:]...)
			return true
		}
	}
	return false
}

//------------------------------------------------------------------------------

func (q *baseQuery) modelHasTableName() bool {
	return !q.modelTable.IsZero() || q.table != nil
}

func (q *baseQuery) hasTables() bool {
	return q.modelHasTableName() || len(q.tables) > 0
}

func (q *baseQuery) appendTables(
	fmter schema.Formatter, b []byte,
) (_ []byte, err error) {
	return q._appendTables(fmter, b, false)
}

func (q *baseQuery) appendTablesWithAlias(
	fmter schema.Formatter, b []byte,
) (_ []byte, err error) {
	return q._appendTables(fmter, b, true)
}

func (q *baseQuery) _appendTables(
	fmter schema.Formatter, b []byte, withAlias bool,
) (_ []byte, err error) {
	startLen := len(b)

	if q.modelHasTableName() {
		if !q.modelTable.IsZero() {
			b, err = q.modelTable.AppendQuery(fmter, b)
			if err != nil {
				return nil, err
			}
		} else {
			b = fmter.AppendQuery(b, string(q.table.SQLNameForSelects))
			if withAlias && q.table.SQLAlias != q.table.SQLNameForSelects {
				b = append(b, " AS "...)
				b = append(b, q.table.SQLAlias...)
			}
		}
	}

	for _, table := range q.tables {
		if len(b) > startLen {
			b = append(b, ", "...)
		}
		b, err = table.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *baseQuery) appendFirstTable(fmter schema.Formatter, b []byte) ([]byte, error) {
	return q._appendFirstTable(fmter, b, false)
}

func (q *baseQuery) appendFirstTableWithAlias(
	fmter schema.Formatter, b []byte,
) ([]byte, error) {
	return q._appendFirstTable(fmter, b, true)
}

func (q *baseQuery) _appendFirstTable(
	fmter schema.Formatter, b []byte, withAlias bool,
) ([]byte, error) {
	if !q.modelTable.IsZero() {
		return q.modelTable.AppendQuery(fmter, b)
	}

	if q.table != nil {
		b = fmter.AppendQuery(b, string(q.table.SQLName))
		if withAlias {
			b = append(b, " AS "...)
			b = append(b, q.table.SQLAlias...)
		}
		return b, nil
	}

	if len(q.tables) > 0 {
		return q.tables[0].AppendQuery(fmter, b)
	}

	return nil, errors.New("bun: query does not have a table")
}

func (q *baseQuery) hasMultiTables() bool {
	if q.modelHasTableName() {
		return len(q.tables) >= 1
	}
	return len(q.tables) >= 2
}

func (q *baseQuery) appendOtherTables(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	tables := q.tables
	if !q.modelHasTableName() {
		tables = tables[1:]
	}
	for i, table := range tables {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = table.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

//------------------------------------------------------------------------------

func (q *baseQuery) appendColumns(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	for i, f := range q.columns {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (q *baseQuery) getFields() ([]*schema.Field, error) {
	table := q.tableModel.Table()

	if len(q.columns) == 0 {
		return table.Fields, nil
	}

	fields, err := q._getFields(false)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func (q *baseQuery) getDataFields() ([]*schema.Field, error) {
	if len(q.columns) == 0 {
		return q.table.DataFields, nil
	}
	return q._getFields(true)
}

func (q *baseQuery) _getFields(omitPK bool) ([]*schema.Field, error) {
	fields := make([]*schema.Field, 0, len(q.columns))
	for _, col := range q.columns {
		if col.Args != nil {
			continue
		}

		field, err := q.table.Field(col.Query)
		if err != nil {
			return nil, err
		}

		if omitPK && field.IsPK {
			continue
		}

		fields = append(fields, field)
	}
	return fields, nil
}

func (q *baseQuery) scan(
	ctx context.Context,
	queryApp schema.QueryAppender,
	query string,
	dest []interface{},
) (res result, _ error) {
	ctx, event := q.db.beforeQuery(ctx, queryApp, query, nil)

	rows, err := q.dbi.QueryContext(ctx, query)
	if err != nil {
		q.db.afterQuery(ctx, event, nil, err)
		return res, err
	}
	defer rows.Close()

	model, err := q.getModel(dest)
	if err != nil {
		q.db.afterQuery(ctx, event, nil, err)
		return res, err
	}

	n, err := model.ScanRows(ctx, rows)
	if err != nil {
		q.db.afterQuery(ctx, event, nil, err)
		return res, err
	}
	res.n = n

	q.db.afterQuery(ctx, event, nil, err)
	return res, nil
}

func (q *baseQuery) exec(
	ctx context.Context,
	queryApp schema.QueryAppender,
	query string,
) (res result, _ error) {
	ctx, event := q.db.beforeQuery(ctx, queryApp, query, nil)

	r, err := q.dbi.ExecContext(ctx, query)
	if err != nil {
		q.db.afterQuery(ctx, event, nil, err)
		return res, err
	}

	res.r = r

	q.db.afterQuery(ctx, event, nil, err)
	return res, nil
}

//------------------------------------------------------------------------------

func (q *baseQuery) AppendArg(fmter schema.Formatter, b []byte, name string) ([]byte, bool) {
	if q.table == nil {
		return b, false
	}

	switch name {
	case "TableName":
		b = fmter.AppendQuery(b, string(q.table.SQLName))
		return b, true
	case "TableAlias":
		b = fmter.AppendQuery(b, string(q.table.SQLAlias))
		return b, true
	case "PKs":
		b = appendColumns(b, "", q.table.PKs)
		return b, true
	case "TablePKs":
		b = appendColumns(b, q.table.SQLAlias, q.table.PKs)
		return b, true
	case "Columns":
		b = appendColumns(b, "", q.table.Fields)
		return b, true
	case "TableColumns":
		b = appendColumns(b, q.table.SQLAlias, q.table.Fields)
		return b, true
	}

	return b, false
}

func appendColumns(b []byte, table schema.Safe, fields []*schema.Field) []byte {
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}

		if len(table) > 0 {
			b = append(b, table...)
			b = append(b, '.')
		}
		b = append(b, f.SQLName...)
	}
	return b
}

func formatterWithModel(
	fmter schema.Formatter, model schema.ArgAppender,
) schema.Formatter {
	if fmter.IsNop() {
		return fmter
	}
	return fmter.WithModel(model)
}

//------------------------------------------------------------------------------

type WhereQuery struct {
	where []schema.QueryWithSep
}

func (q *WhereQuery) Where(query string, args ...interface{}) *WhereQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *WhereQuery) WhereOr(query string, args ...interface{}) *WhereQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " OR "))
	return q
}

func (q *WhereQuery) addWhere(where schema.QueryWithSep) {
	q.where = append(q.where, where)
}

func (q *WhereQuery) WhereGroup(sep string, fn func(*WhereQuery)) {
	q.addWhereGroup(sep, fn)
}

func (q *WhereQuery) addWhereGroup(sep string, fn func(*WhereQuery)) {
	q2 := new(WhereQuery)
	fn(q2)

	if len(q2.where) > 0 {
		q2.where[0].Sep = ""

		q.addWhere(schema.SafeQueryWithSep("", nil, sep+"("))
		q.where = append(q.where, q2.where...)
		q.addWhere(schema.SafeQueryWithSep("", nil, ")"))
	}
}

//------------------------------------------------------------------------------

type whereBaseQuery struct {
	baseQuery
	WhereQuery
}

func (q *whereBaseQuery) mustAppendWhere(fmter schema.Formatter, b []byte) ([]byte, error) {
	if len(q.where) == 0 && !q.flags.Has(wherePKFlag) {
		err := errors.New(
			"bun: Update and Delete queries require Where clause (try WherePK)")
		return nil, err
	}
	return q.appendWhere(fmter, b)
}

func (q *whereBaseQuery) appendWhere(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if len(q.where) == 0 && !q.isSoftDelete() && !q.flags.Has(wherePKFlag) {
		return b, nil
	}

	b = append(b, " WHERE "...)
	startLen := len(b)

	if len(q.where) > 0 {
		b, err = q._appendWhere(fmter, b, q.where)
		if err != nil {
			return nil, err
		}
	}

	if q.isSoftDelete() {
		if len(b) > startLen {
			b = append(b, " AND "...)
		}
		b = append(b, q.tableModel.Table().SQLAlias...)
		b = q.appendWhereSoftDelete(b)
	}

	if q.flags.Has(wherePKFlag) {
		if len(b) > startLen {
			b = append(b, " AND "...)
		}
		b, err = q.appendWherePK(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *whereBaseQuery) _appendWhere(
	fmter schema.Formatter, b []byte, where []schema.QueryWithSep,
) (_ []byte, err error) {
	for i, where := range where {
		if i > 0 {
			b = append(b, where.Sep...)
		}

		if where.Query == "" && where.Args == nil {
			continue
		}

		b = append(b, '(')
		b, err = where.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
		b = append(b, ')')
	}
	return b, nil
}

func (q *whereBaseQuery) appendWhereSoftDelete(b []byte) []byte {
	b = append(b, '.')
	b = append(b, q.tableModel.Table().SoftDeleteField.SQLName...)
	if q.flags.Has(deletedFlag) {
		b = append(b, " IS NOT NULL"...)
	} else {
		b = append(b, " IS NULL"...)
	}
	return b
}

func (q *whereBaseQuery) appendWherePK(
	fmter schema.Formatter, b []byte,
) (_ []byte, err error) {
	if err := q.table.CheckPKs(); err != nil {
		return nil, err
	}

	switch model := q.tableModel.(type) {
	case *structTableModel:
		return q.appendWherePKStruct(fmter, b, model)
	case *sliceTableModel:
		return q.appendWherePKSlice(fmter, b, model)
	}

	return nil, fmt.Errorf("bun: WherePK does not support %T", q.tableModel)
}

func (q *whereBaseQuery) appendWherePKStruct(
	fmter schema.Formatter, b []byte, model *structTableModel,
) (_ []byte, err error) {
	isTemplate := fmter.IsNop()
	b = append(b, '(')
	for i, f := range q.table.PKs {
		if i > 0 {
			b = append(b, " AND "...)
		}
		b = append(b, q.table.SQLAlias...)
		b = append(b, '.')
		b = append(b, f.SQLName...)
		b = append(b, " = "...)
		if isTemplate {
			b = append(b, '?')
		} else {
			b = f.AppendValue(fmter, b, model.strct)
		}
	}
	b = append(b, ')')
	return b, nil
}

func (q *whereBaseQuery) appendWherePKSlice(
	fmter schema.Formatter, b []byte, model *sliceTableModel,
) (_ []byte, err error) {
	if len(q.table.PKs) > 1 {
		b = append(b, '(')
	}
	b = appendColumns(b, q.table.SQLAlias, q.table.PKs)
	if len(q.table.PKs) > 1 {
		b = append(b, ')')
	}

	b = append(b, " IN ("...)

	isTemplate := fmter.IsNop()
	slice := model.slice
	sliceLen := slice.Len()
	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			if isTemplate {
				break
			}
			b = append(b, ", "...)
		}

		el := indirect(slice.Index(i))

		if len(q.table.PKs) > 1 {
			b = append(b, '(')
		}
		for i, f := range q.table.PKs {
			if i > 0 {
				b = append(b, ", "...)
			}
			if isTemplate {
				b = append(b, '?')
			} else {
				b = f.AppendValue(fmter, b, el)
			}
		}
		if len(q.table.PKs) > 1 {
			b = append(b, ')')
		}
	}

	b = append(b, ')')

	return b, nil
}

//------------------------------------------------------------------------------

type returningQuery struct {
	returning       []schema.QueryWithArgs
	returningFields []*schema.Field
}

func (q *returningQuery) addReturning(ret schema.QueryWithArgs) {
	q.returning = append(q.returning, ret)
}

func (q *returningQuery) addReturningField(field *schema.Field) {
	if len(q.returning) > 0 {
		return
	}
	for _, f := range q.returningFields {
		if f == field {
			return
		}
	}
	q.returningFields = append(q.returningFields, field)
}

func (q *returningQuery) hasReturning() bool {
	if len(q.returning) == 1 {
		switch q.returning[0].Query {
		case "null", "NULL":
			return false
		}
	}
	return len(q.returning) > 0 || len(q.returningFields) > 0
}

func (q *returningQuery) appendReturning(
	fmter schema.Formatter, b []byte,
) (_ []byte, err error) {
	if !q.hasReturning() {
		return b, nil
	}

	b = append(b, " RETURNING "...)

	for i, f := range q.returning {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	if len(q.returning) > 0 {
		return b, nil
	}

	b = appendColumns(b, "", q.returningFields)
	return b, nil
}

//------------------------------------------------------------------------------

type columnValue struct {
	column string
	value  schema.QueryWithArgs
}

type customValueQuery struct {
	modelValues map[string]schema.QueryWithArgs
	extraValues []columnValue
}

func (q *customValueQuery) addValue(
	table *schema.Table, column string, value string, args []interface{},
) {
	if _, ok := table.FieldMap[column]; ok {
		if q.modelValues == nil {
			q.modelValues = make(map[string]schema.QueryWithArgs)
		}
		q.modelValues[column] = schema.SafeQuery(value, args)
	} else {
		q.extraValues = append(q.extraValues, columnValue{
			column: column,
			value:  schema.SafeQuery(value, args),
		})
	}
}

//------------------------------------------------------------------------------

type setQuery struct {
	set []schema.QueryWithArgs
}

func (q *setQuery) addSet(set schema.QueryWithArgs) {
	q.set = append(q.set, set)
}

func (q setQuery) appendSet(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, " SET "...)
	for i, f := range q.set {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

//------------------------------------------------------------------------------

type cascadeQuery struct {
	restrict bool
}

func (q cascadeQuery) appendCascade(fmter schema.Formatter, b []byte) []byte {
	if !fmter.HasFeature(feature.TableCascade) {
		return b
	}
	if q.restrict {
		b = append(b, " RESTRICT"...)
	} else {
		b = append(b, " CASCADE"...)
	}
	return b
}

//------------------------------------------------------------------------------

var byteSlicePool = sync.Pool{
	New: func() interface{} {
		return &byteSlice{
			b: make([]byte, 0, 4096),
		}
	},
}

type byteSlice struct {
	b []byte
}

func getByteSlice() *byteSlice {
	return byteSlicePool.Get().(*byteSlice)
}

func putByteSlice(b *byteSlice) {
	b.b = b.b[:0]
	byteSlicePool.Put(b)
}

//------------------------------------------------------------------------------

func errNoRows(rows *sql.Rows) error {
	if err := rows.Err(); err != nil {
		return err
	}
	return sql.ErrNoRows
}
