package bun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun/dialect"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

// UpdateQuery builds SQL UPDATE statements.
type UpdateQuery struct {
	whereBaseQuery
	orderLimitOffsetQuery
	returningQuery
	setQuery
	idxHintsQuery

	joins   []joinQuery
	comment string
}

var _ Query = (*UpdateQuery)(nil)

// NewUpdateQuery returns a new UpdateQuery attached to the provided DB.
func NewUpdateQuery(db *DB) *UpdateQuery {
	q := &UpdateQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db: db,
			},
		},
	}
	return q
}

// Conn sets the database connection for this query.
func (q *UpdateQuery) Conn(db IConn) *UpdateQuery {
	q.setConn(db)
	return q
}

// Model sets the model whose rows are updated and provides the values for the SET clause.
func (q *UpdateQuery) Model(model any) *UpdateQuery {
	q.setModel(model)
	return q
}

// Err sets an error on the query, causing subsequent operations to fail.
func (q *UpdateQuery) Err(err error) *UpdateQuery {
	q.setErr(err)
	return q
}

// Apply calls each function in fns, passing the UpdateQuery as an argument.
func (q *UpdateQuery) Apply(fns ...func(*UpdateQuery) *UpdateQuery) *UpdateQuery {
	for _, fn := range fns {
		if fn != nil {
			q = fn(q)
		}
	}
	return q
}

// With adds a WITH clause (Common Table Expression) to the query.
func (q *UpdateQuery) With(name string, query Query) *UpdateQuery {
	q.addWith(NewWithQuery(name, query))
	return q
}

// WithRecursive adds a WITH RECURSIVE clause to the query.
func (q *UpdateQuery) WithRecursive(name string, query Query) *UpdateQuery {
	q.addWith(NewWithQuery(name, query).Recursive())
	return q
}

// WithQuery adds a pre-configured WITH clause to the query.
func (q *UpdateQuery) WithQuery(query *WithQuery) *UpdateQuery {
	q.addWith(query)
	return q
}

// ------------------------------------------------------------------------------

// Table adds table(s) to the query.
//
// If the query has no model, the first table becomes the update target.
// How the other tables are rendered depends on the dialect:
//   - Dialects that support multi-table UPDATE (MySQL) list all tables
//     after UPDATE.
//   - Other dialects (e.g. PostgreSQL, MSSQL) emit the remaining tables in
//     the FROM clause; they can be used in WHERE/SET but are not updated.
func (q *UpdateQuery) Table(tables ...string) *UpdateQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

// TableExpr adds a table expression with arguments to the query.
func (q *UpdateQuery) TableExpr(query string, args ...any) *UpdateQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

// ModelTableExpr overrides the table name derived from the model.
func (q *UpdateQuery) ModelTableExpr(query string, args ...any) *UpdateQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

// TableAlias overrides the alias used for the model's table in this query.
func (q *UpdateQuery) TableAlias(alias string) *UpdateQuery {
	q.setTableAlias(alias)
	return q
}

//------------------------------------------------------------------------------

// Column restricts the SET clause to the given model columns.
func (q *UpdateQuery) Column(columns ...string) *UpdateQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

// ExcludeColumn excludes specific model columns from the SET clause.
func (q *UpdateQuery) ExcludeColumn(columns ...string) *UpdateQuery {
	q.excludeColumn(columns)
	return q
}

// Set adds an expression to the SET clause with optional arguments.
func (q *UpdateQuery) Set(query string, args ...any) *UpdateQuery {
	q.addSet(schema.SafeQuery(query, args))
	return q
}

// SetColumn adds a "column = expression" assignment to the SET clause.
// On dialects that support multi-table UPDATE, the column is qualified with the model table alias.
func (q *UpdateQuery) SetColumn(column string, query string, args ...any) *UpdateQuery {
	if q.db.HasFeature(feature.UpdateMultiTable) {
		column = q.table.Alias + "." + column
	}
	q.addSet(schema.SafeQuery(column+" = "+query, args))
	return q
}

// Value overwrites model value for the column.
func (q *UpdateQuery) Value(column string, query string, args ...any) *UpdateQuery {
	if q.table == nil {
		q.setErr(errNilModel)
		return q
	}
	q.addValue(q.table, column, query, args)
	return q
}

// OmitZero omits model fields that hold zero values from the SET clause.
func (q *UpdateQuery) OmitZero() *UpdateQuery {
	q.omitZero = true
	return q
}

//------------------------------------------------------------------------------

// Join adds a JOIN clause with the specified join expression.
func (q *UpdateQuery) Join(join string, args ...any) *UpdateQuery {
	q.joins = append(q.joins, joinQuery{
		join: schema.SafeQuery(join, args),
	})
	return q
}

// JoinOn adds an ON condition to the most recent JOIN, combined with AND.
func (q *UpdateQuery) JoinOn(cond string, args ...any) *UpdateQuery {
	return q.joinOn(cond, args, " AND ")
}

// JoinOnOr adds an ON condition to the most recent JOIN, combined with OR.
func (q *UpdateQuery) JoinOnOr(cond string, args ...any) *UpdateQuery {
	return q.joinOn(cond, args, " OR ")
}

func (q *UpdateQuery) joinOn(cond string, args []any, sep string) *UpdateQuery {
	if len(q.joins) == 0 {
		q.setErr(errors.New("bun: query has no joins"))
		return q
	}
	j := &q.joins[len(q.joins)-1]
	j.on = append(j.on, schema.SafeQueryWithSep(cond, args, sep))
	return q
}

// JoinModel adds a JOIN clause on the model's table, using alias as the
// table alias or the model's table alias when alias is omitted.
//
// If the model's table has a soft-delete field, an `alias.field IS NULL`
// condition is appended to the join's ON clause. Use WhereAllWithDeleted
// to include soft-deleted rows of the joined table.
func (q *UpdateQuery) JoinModel(model any, alias ...string) *UpdateQuery {
	return q.joinModel("JOIN", model, alias)
}

// LeftJoinModel is like JoinModel but emits a LEFT JOIN clause.
func (q *UpdateQuery) LeftJoinModel(model any, alias ...string) *UpdateQuery {
	return q.joinModel("LEFT JOIN", model, alias)
}

func (q *UpdateQuery) joinModel(kind string, model any, alias []string) *UpdateQuery {
	j, err := newModelJoinQuery(q.db, kind, model, alias)
	if err != nil {
		q.setErr(err)
		return q
	}
	q.joins = append(q.joins, j)
	return q
}

//------------------------------------------------------------------------------

// WherePK adds a WHERE condition on the model's primary key columns.
// When cols are provided, those columns are used instead of the primary key.
func (q *UpdateQuery) WherePK(cols ...string) *UpdateQuery {
	q.addWhereCols(cols)
	return q
}

// Where adds a WHERE condition combined with AND.
func (q *UpdateQuery) Where(query string, args ...any) *UpdateQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " AND "))
	return q
}

// WhereOr adds a WHERE condition combined with OR.
func (q *UpdateQuery) WhereOr(query string, args ...any) *UpdateQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " OR "))
	return q
}

// WhereGroup groups WHERE conditions added by fn with the given separator (AND/OR).
func (q *UpdateQuery) WhereGroup(sep string, fn func(*UpdateQuery) *UpdateQuery) *UpdateQuery {
	saved, savedHasOr := q.where, q.whereHasOr
	q.where, q.whereHasOr = nil, false

	q = fn(q)

	where := q.where
	q.where, q.whereHasOr = saved, savedHasOr

	q.addWhereGroup(sep, where)

	return q
}

// WhereDeleted adds a WHERE condition to update soft-deleted rows only.
func (q *UpdateQuery) WhereDeleted() *UpdateQuery {
	q.whereDeleted()
	return q
}

// WhereAllWithDeleted includes both active and soft-deleted rows.
func (q *UpdateQuery) WhereAllWithDeleted() *UpdateQuery {
	q.whereAllWithDeleted()
	return q
}

// ------------------------------------------------------------------------------

// Order adds an ORDER BY clause with the given columns.
// It requires a dialect that supports ORDER BY in UPDATE statements and
// otherwise sets feature.NewNotSupportError on the query.
func (q *UpdateQuery) Order(orders ...string) *UpdateQuery {
	if !q.hasFeature(feature.UpdateOrderLimit) {
		q.setErr(feature.NewNotSupportError(feature.UpdateOrderLimit))
		return q
	}
	q.addOrder(orders...)
	return q
}

// OrderExpr adds an ORDER BY expression with optional arguments.
// It has the same dialect requirement as Order.
func (q *UpdateQuery) OrderExpr(query string, args ...any) *UpdateQuery {
	if !q.hasFeature(feature.UpdateOrderLimit) {
		q.setErr(feature.NewNotSupportError(feature.UpdateOrderLimit))
		return q
	}
	q.addOrderExpr(query, args...)
	return q
}

// Limit sets the maximum number of rows to update.
// It requires a dialect that supports LIMIT in UPDATE statements and
// otherwise sets feature.NewNotSupportError on the query.
func (q *UpdateQuery) Limit(n int64) *UpdateQuery {
	if !q.hasFeature(feature.UpdateOrderLimit) {
		q.setErr(feature.NewNotSupportError(feature.UpdateOrderLimit))
		return q
	}
	q.setLimit(n)
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("NULL")`.
func (q *UpdateQuery) Returning(query string, args ...any) *UpdateQuery {
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *UpdateQuery) Comment(comment string) *UpdateQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

// Operation returns the query operation name ("UPDATE").
func (q *UpdateQuery) Operation() string {
	return "UPDATE"
}

// AppendQuery appends the generated UPDATE statement to b.
func (q *UpdateQuery) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	gen = formatterWithModel(gen, q)

	b, err = q.appendWith(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, "UPDATE "...)

	if gen.HasFeature(feature.UpdateMultiTable) {
		b, err = q.appendTablesWithAlias(gen, b)
	} else if gen.HasFeature(feature.UpdateTableAlias) || q.hasExplicitTableAlias() {
		b, err = q.appendFirstTableWithAlias(gen, b)
	} else {
		b, err = q.appendFirstTable(gen, b)
	}
	if err != nil {
		return nil, err
	}

	b, err = q.appendIndexHints(gen, b)
	if err != nil {
		return nil, err
	}

	b, err = q.mustAppendSet(gen, b)
	if err != nil {
		return nil, err
	}

	if !gen.HasFeature(feature.UpdateMultiTable) {
		b, err = q.appendOtherTables(gen, b)
		if err != nil {
			return nil, err
		}
	}

	for _, j := range q.joins {
		b, err = j.AppendQuery(gen, b, q.flags)
		if err != nil {
			return nil, err
		}
	}

	if q.hasFeature(feature.Output) && q.hasReturning() {
		b = append(b, " OUTPUT "...)
		b, err = q.appendOutput(gen, b)
		if err != nil {
			return nil, err
		}
	}

	b, err = q.mustAppendWhere(gen, b, q.hasTableAlias(gen))
	if err != nil {
		return nil, err
	}

	b, err = q.appendOrder(gen, b)
	if err != nil {
		return nil, err
	}

	b, err = q.appendLimitOffset(gen, b)
	if err != nil {
		return nil, err
	}

	if q.hasFeature(feature.Returning) && q.hasReturning() {
		b = append(b, " RETURNING "...)
		b, err = q.appendReturning(gen, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *UpdateQuery) mustAppendSet(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	b = append(b, " SET "...)
	pos := len(b)

	switch model := q.model.(type) {
	case *structTableModel:
		if !model.strct.IsValid() { // Model((*Foo)(nil))
			break
		}
		if len(q.set) > 0 && q.columns == nil {
			break
		}

		fields, err := q.getDataFields()
		if err != nil {
			return nil, err
		}

		b, err = q.appendSetStruct(gen, b, model, fields)
		if err != nil {
			return nil, err
		}

	case *sliceTableModel:
		if len(q.set) > 0 { // bulk-update
			return q.appendSet(gen, b)
		}
		return nil, errors.New("bun: to bulk Update, use CTE and VALUES")

	case *mapModel:
		b = model.appendSet(gen, b)

	case nil:
		// continue below

	default:
		return nil, fmt.Errorf("bun: Update does not support %T", q.model)
	}

	if len(q.set) > 0 {
		if len(b) > pos {
			b = append(b, ", "...)
		}
		return q.appendSet(gen, b)
	}

	if len(b) == pos {
		return nil, errors.New("bun: empty SET clause is not allowed in the UPDATE query")
	}
	return b, nil
}

func (q *UpdateQuery) appendOtherTables(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if !q.hasMultiTables() {
		return b, nil
	}

	b = append(b, " FROM "...)

	b, err = q.whereBaseQuery.appendOtherTables(gen, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//------------------------------------------------------------------------------

// Bulk rewrites the query to update multiple rows from a slice model,
// joining the model table against the slice values exposed as a WITH clause.
// The model must be a slice, otherwise an error is set on the query.
func (q *UpdateQuery) Bulk() *UpdateQuery {
	model, ok := q.model.(*sliceTableModel)
	if !ok {
		q.setErr(fmt.Errorf("bun: Bulk requires a slice, got %T", q.model))
		return q
	}

	set, err := q.updateSliceSet(q.db.gen, model)
	if err != nil {
		q.setErr(err)
		return q
	}

	values := q.db.NewValues(model)
	values.customValueQuery = q.customValueQuery

	return q.With("_data", values).
		Model(model).
		TableExpr("_data").
		Set(set).
		Where(q.updateSliceWhere(q.db.gen, model))
}

func (q *UpdateQuery) updateSliceSet(
	gen schema.QueryGen, model *sliceTableModel,
) (string, error) {
	fields, err := q.getDataFields()
	if err != nil {
		return "", err
	}

	var b []byte
	pos := len(b)
	for _, field := range fields {
		if field.SkipUpdate() {
			continue
		}
		if len(b) != pos {
			b = append(b, ", "...)
			pos = len(b)
		}
		if gen.HasFeature(feature.UpdateMultiTable) {
			b = append(b, model.table.SQLAlias...)
			b = append(b, '.')
		}
		b = append(b, field.SQLName...)
		b = append(b, " = _data."...)
		b = append(b, field.SQLName...)
	}
	return internal.String(b), nil
}

func (q *UpdateQuery) updateSliceWhere(gen schema.QueryGen, model *sliceTableModel) string {
	var b []byte
	for i, pk := range model.table.PKs {
		if i > 0 {
			b = append(b, " AND "...)
		}
		if q.hasTableAlias(gen) {
			b = append(b, model.table.SQLAlias...)
		} else {
			b = append(b, model.table.SQLName...)
		}
		b = append(b, '.')
		b = append(b, pk.SQLName...)
		b = append(b, " = _data."...)
		b = append(b, pk.SQLName...)
	}
	return internal.String(b)
}

//------------------------------------------------------------------------------

// Scan executes the UPDATE and scans RETURNING/OUTPUT results into dest.
func (q *UpdateQuery) Scan(ctx context.Context, dest ...any) error {
	_, err := q.scanOrExec(ctx, dest, true)
	return err
}

// Exec executes the UPDATE and optionally scans RETURNING/OUTPUT results into dest when provided.
func (q *UpdateQuery) Exec(ctx context.Context, dest ...any) (sql.Result, error) {
	return q.scanOrExec(ctx, dest, len(dest) > 0)
}

func (q *UpdateQuery) scanOrExec(
	ctx context.Context, dest []any, hasDest bool,
) (sql.Result, error) {
	if q.err != nil {
		return nil, q.err
	}

	if q.table != nil {
		if err := q.beforeUpdateHook(ctx); err != nil {
			return nil, err
		}
	}

	// Run append model hooks before generating the query.
	if err := q.beforeAppendModel(ctx, q); err != nil {
		return nil, err
	}

	// if a comment is propagated via the context, use it
	setCommentFromContext(ctx, q)

	// Generate the query before checking hasReturning.
	queryBytes, err := q.AppendQuery(q.db.gen, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	useScan := hasDest || (q.hasReturning() && q.hasFeature(feature.Returning|feature.Output))
	var model Model

	if useScan {
		var err error
		model, err = q.getModel(dest)
		if err != nil {
			return nil, err
		}
	}

	query := internal.String(queryBytes)

	var res sql.Result

	if useScan {
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

// FQN returns a fully qualified column name, for example, table_name.column_name or
// table_alias.column_alias.
func (q *UpdateQuery) FQN(column string) Ident {
	if q.table == nil {
		panic("UpdateQuery.FQN requires a model")
	}
	if q.hasTableAlias(q.db.gen) {
		return Ident(q.table.Alias + "." + column)
	}
	return Ident(q.table.Name + "." + column)
}

func (q *UpdateQuery) hasTableAlias(gen schema.QueryGen) bool {
	return gen.HasFeature(feature.UpdateMultiTable|feature.UpdateTableAlias) ||
		q.hasExplicitTableAlias()
}

// String returns the generated SQL query string. The UpdateQuery instance must not be
// modified during query generation to ensure multiple calls to String() return identical results.
func (q *UpdateQuery) String() string {
	buf, err := q.AppendQuery(q.db.QueryGen(), nil)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

//------------------------------------------------------------------------------

// QueryBuilder wraps the UpdateQuery in a generic QueryBuilder interface.
func (q *UpdateQuery) QueryBuilder() QueryBuilder {
	return &updateQueryBuilder{q}
}

// ApplyQueryBuilder applies fn to a generic QueryBuilder and returns the modified UpdateQuery.
func (q *UpdateQuery) ApplyQueryBuilder(fn func(QueryBuilder) QueryBuilder) *UpdateQuery {
	return fn(q.QueryBuilder()).Unwrap().(*UpdateQuery)
}

type updateQueryBuilder struct {
	*UpdateQuery
}

func (q *updateQueryBuilder) WhereGroup(
	sep string, fn func(QueryBuilder) QueryBuilder,
) QueryBuilder {
	q.UpdateQuery = q.UpdateQuery.WhereGroup(sep, func(qs *UpdateQuery) *UpdateQuery {
		return fn(q).(*updateQueryBuilder).UpdateQuery
	})
	return q
}

func (q *updateQueryBuilder) Where(query string, args ...any) QueryBuilder {
	q.UpdateQuery.Where(query, args...)
	return q
}

func (q *updateQueryBuilder) WhereOr(query string, args ...any) QueryBuilder {
	q.UpdateQuery.WhereOr(query, args...)
	return q
}

func (q *updateQueryBuilder) WhereDeleted() QueryBuilder {
	q.UpdateQuery.WhereDeleted()
	return q
}

func (q *updateQueryBuilder) WhereAllWithDeleted() QueryBuilder {
	q.UpdateQuery.WhereAllWithDeleted()
	return q
}

func (q *updateQueryBuilder) WherePK(cols ...string) QueryBuilder {
	q.UpdateQuery.WherePK(cols...)
	return q
}

func (q *updateQueryBuilder) Unwrap() any {
	return q.UpdateQuery
}

//------------------------------------------------------------------------------

// UseIndex adds a USE INDEX hint for MySQL to suggest index usage.
func (q *UpdateQuery) UseIndex(indexes ...string) *UpdateQuery {
	if q.db.dialect.Name() == dialect.MySQL {
		q.addUseIndex(indexes...)
	}
	return q
}

// IgnoreIndex adds an IGNORE INDEX hint for MySQL to prevent index usage.
func (q *UpdateQuery) IgnoreIndex(indexes ...string) *UpdateQuery {
	if q.db.dialect.Name() == dialect.MySQL {
		q.addIgnoreIndex(indexes...)
	}
	return q
}

// ForceIndex adds a FORCE INDEX hint for MySQL to require index usage.
func (q *UpdateQuery) ForceIndex(indexes ...string) *UpdateQuery {
	if q.db.dialect.Name() == dialect.MySQL {
		q.addForceIndex(indexes...)
	}
	return q
}
