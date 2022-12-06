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

// chainableAlterTableSubquery allows doing multiple modifications in a single ALTER TABLE query.
// Structs that implement SubqueryAppender can embed chainableAlterTableSubquery to implement ChainableSubquery too.
type chainableAlterTableSubquery struct {
	parent *AlterTableQuery
}

func (_ *chainableAlterTableSubquery) chain() {}

func (q *chainableAlterTableSubquery) AlterColumn() *AlterColumnSubquery {
	return q.parent.AlterColumn()
}

func (q *chainableAlterTableSubquery) AddColumn() *AddColumnSubquery {
	return q.parent.AddColumn()
}

func (q *chainableAlterTableSubquery) DropColumn() *DropColumnSubquery {
	return q.parent.DropColumn()
}

// ------------------------------------------------------------------------------

type AlterTableQuery struct {
	baseQuery
	ifExists
	subqueries []SubqueryAppender
}

var _ Query = (*AlterTableQuery)(nil)

func NewAlterTableQuery(db *DB) *AlterTableQuery {
	return &AlterTableQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
	}
}

func (q *AlterTableQuery) Operation() string {
	return "ALTER TABLE"
}

func (q *AlterTableQuery) Model(model interface{}) *AlterTableQuery {
	q.setModel(model)
	return q
}

// ------------------------------------------------------------------------------

func (q *AlterTableQuery) Rename() *RenameTableSubquery {
	sq := newRenameTableQuery(q.db, q)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

func (q *AlterTableQuery) RenameColumn() *RenameColumnSubquery {
	sq := newRenameColumnQuery(q.db, q)
	q.subqueries = append(q.subqueries, sq)
	return sq
}

func (q *AlterTableQuery) AlterColumn() *AlterColumnSubquery {
	sq := newAlterColumnQuery(q.db, q)
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

type RenameTableSubquery struct {
	baseQuery
	parent *AlterTableQuery
}

var (
	_ SubqueryAppender = (*RenameTableSubquery)(nil)
)

func newRenameTableQuery(db *DB, parent *AlterTableQuery) *RenameTableSubquery {
	return &RenameTableSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent: parent,
	}
}

func (q *RenameTableSubquery) To(newName string) *RenameTableSubquery {
	q.addColumn(renameQuery("", newName))
	return q
}

func (q *RenameTableSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "RENAME "...)
	b, err = q.appendFirstColumn(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *RenameTableSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// RENAME COLUMN --------------------------------------------------------------

type RenameColumnSubquery struct {
	baseQuery
	parent *AlterTableQuery
	which  string
}

var (
	_ SubqueryAppender = (*RenameColumnSubquery)(nil)
)

func newRenameColumnQuery(db *DB, parent *AlterTableQuery) *RenameColumnSubquery {
	return &RenameColumnSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		parent: parent,
	}
}

func (q *RenameColumnSubquery) Column(column string) *RenameColumnSubquery {
	q.which = column
	return q
}

func (q *RenameColumnSubquery) To(newName string) *RenameColumnSubquery {
	q.addColumn(renameQuery(q.which, newName))
	return q
}

func (q *RenameColumnSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "RENAME COLUMN "...)
	b, err = q.appendFirstColumn(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *RenameColumnSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// ALTER COLUMN ---------------------------------------------------------------

type AlterColumnSubquery struct {
	baseQuery
	chainableAlterTableSubquery
	which        schema.QueryWithArgs
	modification schema.QueryAppender
}

var (
	_ ChainableSubquery = (*AlterColumnSubquery)(nil)
)

func newAlterColumnQuery(db *DB, parent *AlterTableQuery) *AlterColumnSubquery {
	return &AlterColumnSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		chainableAlterTableSubquery: chainableAlterTableSubquery{
			parent: parent,
		},
	}
}

func (q *AlterColumnSubquery) Column(column string) *AlterColumnSubquery {
	q.which = schema.UnsafeIdent(column)
	return q
}

func (q *AlterColumnSubquery) Type(typ string) *AlterColumnSubquery {
	q.modification = schema.QueryWithArgs{
		Query: "SET DATA TYPE ?",
		Args:  []interface{}{schema.Safe(typ)},
	}
	return q
}

func (q *AlterColumnSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "ALTER COLUMN "...)
	b, err = q.which.AppendQuery(fmter, b)
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

func (q *AlterColumnSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// ADD COLUMN -----------------------------------------------------------------

type AddColumnSubquery struct {
	baseQuery
	chainableAlterTableSubquery
	ifNotExists
}

var (
	_ ChainableSubquery = (*AddColumnSubquery)(nil)
)

func newAddColumnSubquery(db *DB, parent *AlterTableQuery) *AddColumnSubquery {
	return &AddColumnSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		chainableAlterTableSubquery: chainableAlterTableSubquery{
			parent: parent,
		},
	}
}

func (q *AddColumnSubquery) ColumnExpr(query string, args ...interface{}) *AddColumnSubquery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

func (q *AddColumnSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "ADD COLUMN "...)
	b, _ = q.ifNotExists.AppendQuery(fmter, b)

	b, err = q.appendFirstColumn(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *AddColumnSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// DROP COLUMN -----------------------------------------------------------------

type DropColumnSubquery struct {
	baseQuery
	chainableAlterTableSubquery
	ifExists
}

var (
	_ ChainableSubquery = (*DropColumnSubquery)(nil)
)

func newDropColumnSubquery(db *DB, parent *AlterTableQuery) *DropColumnSubquery {
	return &DropColumnSubquery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		chainableAlterTableSubquery: chainableAlterTableSubquery{
			parent: parent,
		},
	}
}

func (q *DropColumnSubquery) Column(column string) *DropColumnSubquery {
	q.addColumn(schema.UnsafeIdent(column))
	return q
}

func (q *DropColumnSubquery) ColumnExpr(query string, args ...interface{}) *DropColumnSubquery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

func (q *DropColumnSubquery) IfExists() *DropColumnSubquery {
	q.ifExists = true
	return q
}

func (q *DropColumnSubquery) AppendSubquery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b = append(b, "DROP COLUMN "...)
	b, _ = q.ifExists.AppendQuery(fmter, b)

	b, err = q.appendFirstColumn(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *DropColumnSubquery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.parent.AppendQuery(fmter, b)
}

// ------------------------------------------------------------------------------

// renameQuery is a convenient wrapper to create query of the form `1? TO 2?` with optionally empty 1st argument.
// It can be used in RENAME clauses for various database objects.
func renameQuery(from, to string) schema.QueryWithArgs {
	query, args := "? TO ?", []interface{}{schema.Ident(from), schema.Ident(to)}
	if from == "" {
		query, args = "TO ?", []interface{}{schema.Ident(to)}
	}
	return schema.QueryWithArgs{Query: query, Args: args}
}
