package bun

import "github.com/uptrace/bun/schema"

type ifExists bool

// AppendQuery adds IF EXISTS clause to the query, if its value is true. It always returns a nil error.
func (ifexists ifExists) AppendQuery(_ schema.Formatter, b []byte) ([]byte, error) {
	if !ifexists {
		return b, nil
	}
	return append(b, "IF EXISTS "...), nil
}

type ifNotExists bool

// AppendQuery adds IF NOT EXISTS clause to the query, if its value is true. It always returns a nil error.
func (ifnotexists ifNotExists) AppendQuery(_ schema.Formatter, b []byte) ([]byte, error) {
	if !ifnotexists {
		return b, nil
	}
	return append(b, "IF NOT EXISTS "...), nil
}

type SubqueryAppender interface {
	schema.QueryAppender
	AppendSubquery(schema.Formatter, []byte) ([]byte, error)
}

type ChainableSubquery interface {
	SubqueryAppender
	chain()
}

type AlterTableQuery struct {
	baseQuery
	ifExists
	subqueries []SubqueryAppender
}

var _ schema.QueryAppender = (*AlterTableQuery)(nil)

func NewAlterTableQuery(db *DB) *AlterTableQuery {
	return &AlterTableQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
	}
}

func (q *AlterTableQuery) Model(model interface{}) *AlterTableQuery {
	q.setModel(model)
	return q
}

// ------------------------------------------------------------------------------

func (q *AlterTableQuery) Rename(to string) *RenameTableQuery {
	sq := newRenameTableQuery(q.db, q, to)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

func (q *AlterTableQuery) RenameColumn(column, to string) *RenameColumnQuery {
	sq := newRenameColumnQuery(q.db, q, column, to)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

func (q *AlterTableQuery) AlterColumn(column string) *AlterColumnQuery {
	sq := newAlterColumnQuery(q.db, q, column)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

func (q *AlterTableQuery) AddColumn() *AddColumnSubquery {
	sq := newAddColumnSubquery(q.db, q)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

func (q *AlterTableQuery) DropColumn() *DropColumnSubquery {
	sq := newDropColumnSubquery(q.db, q)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

// ------------------------------------------------------------------------------

func (q *AlterTableQuery) IfExists() *AlterTableQuery {
	q.ifExists = true
	return q
}

// ------------------------------------------------------------------------------

func (q *AlterTableQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = append(b, "ALTER TABLE "...)
	b, _ = q.ifExists.AppendQuery(fmter, b)

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	if len(q.subqueries) == 0 {
		return b, nil
	}
	b = append(b, " "...)

	if _, chainable := q.subqueries[0].(ChainableSubquery); !chainable {
		return q.subqueries[0].AppendSubquery(fmter, b)
	}

	for i, sub := range q.subqueries {
		if i > 0 {
			b = append(b, ", "...)
		}

		chainable := sub.(ChainableSubquery)
		b, err = chainable.AppendSubquery(fmter, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

// RENAME TO ------------------------------------------------------------------

type RenameTableQuery struct {
	baseQuery
	parent  *AlterTableQuery
	newName schema.QueryWithArgs
}

var (
	_ schema.QueryAppender = (*RenameTableQuery)(nil)
)

func newRenameTableQuery(db *DB, parent *AlterTableQuery, newName string) *RenameTableQuery {
	return &RenameTableQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent:  parent,
		newName: renameQuery("", newName),
	}
}

func (q *RenameTableQuery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "RENAME "...)
	b, err = q.newName.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *RenameTableQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// RENAME COLUMN --------------------------------------------------------------

type RenameColumnQuery struct {
	baseQuery
	parent  *AlterTableQuery
	newName schema.QueryWithArgs
}

var (
	_ schema.QueryAppender = (*RenameColumnQuery)(nil)
)

func newRenameColumnQuery(db *DB, parent *AlterTableQuery, oldName, newName string) *RenameColumnQuery {
	return &RenameColumnQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent:  parent,
		newName: renameQuery(oldName, newName),
	}
}

func (q *RenameColumnQuery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "RENAME COLUMN "...)
	b, err = q.newName.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *RenameColumnQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// ALTER COLUMN ---------------------------------------------------------------

type AlterColumnQuery struct {
	baseQuery
	parent       *AlterTableQuery
	column       schema.QueryWithArgs
	modification schema.QueryAppender
}

var (
	_ ChainableSubquery = (*AlterColumnQuery)(nil)
)

func newAlterColumnQuery(db *DB, parent *AlterTableQuery, column string) *AlterColumnQuery {
	return &AlterColumnQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent: parent,
		column: schema.UnsafeIdent(column),
	}
}

func (q *AlterColumnQuery) AlterColumn(column string) *AlterColumnQuery {
	return q.parent.AlterColumn(column)
}

func (q *AlterColumnQuery) Type(typ string) *AlterColumnQuery {
	q.modification = schema.QueryWithArgs{
		Query: "SET DATA TYPE ?",
		Args:  []interface{}{schema.Safe(typ)},
	}
	return q
}

func (q *AlterColumnQuery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "ALTER COLUMN "...)
	b, err = q.column.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " "...)
	b, err = q.modification.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *AlterColumnQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

func (q *AlterColumnQuery) chain() {}

// ADD COLUMN -----------------------------------------------------------------

type AddColumnSubquery struct {
	baseQuery
	ifNotExists
	column schema.QueryAppender
	parent *AlterTableQuery
}

func newAddColumnSubquery(db *DB, parent *AlterTableQuery) *AddColumnSubquery {
	return &AddColumnSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent: parent,
	}
}

func (q *AddColumnSubquery) ColumnExpr(query string, args ...interface{}) *AddColumnSubquery {
	q.column = schema.SafeQuery(query, args)
	return q
}

func (q *AddColumnSubquery) AddColumn() *AddColumnSubquery {
	return q.parent.AddColumn()
}

func (q *AddColumnSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "ADD COLUMN "...)
	b, _ = q.ifNotExists.AppendQuery(fmter, b)

	b, err = q.column.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *AddColumnSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

func (q *AddColumnSubquery) chain() {}

// DROP COLUMN -----------------------------------------------------------------

type DropColumnSubquery struct {
	baseQuery
	ifExists
	column schema.QueryAppender
	parent *AlterTableQuery
}

func newDropColumnSubquery(db *DB, parent *AlterTableQuery) *DropColumnSubquery {
	return &DropColumnSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent: parent,
	}
}

func (q *DropColumnSubquery) Column(column string) *DropColumnSubquery {
	q.column = schema.UnsafeIdent(column)
	return q
}

func (q *DropColumnSubquery) ColumnExpr(query string, args ...interface{}) *DropColumnSubquery {
	q.column = schema.SafeQuery(query, args)
	return q
}

func (q *DropColumnSubquery) IfExists() *DropColumnSubquery {
	q.ifExists = true
	return q
}

func (q *DropColumnSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "DROP COLUMN "...)
	b, _ = q.ifExists.AppendQuery(fmter, b)

	b, err = q.column.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *DropColumnSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

func (q *DropColumnSubquery) chain() {}

// ------------------------------------------------------------------------------

func renameQuery(from, to string) schema.QueryWithArgs {
	query, args := "? TO ?", []interface{}{schema.Ident(from), schema.Ident(to)}
	if from == "" {
		query, args = "TO ?", []interface{}{schema.Ident(to)}
	}
	return schema.QueryWithArgs{Query: query, Args: args}
}
