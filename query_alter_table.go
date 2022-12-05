package bun

import "github.com/uptrace/bun/schema"

type ifExists bool

func (ifexists ifExists) AppendQuery(_ schema.Formatter, b []byte) ([]byte, error) {
	if !ifexists {
		return b, nil
	}
	return append(b, "IF EXISTS "...), nil
}

type ifExistsAppender interface {
	AppendIfExists(schema.Formatter, []byte) ([]byte, error)
}

type AlterTableQuery struct {
	baseQuery
	subquery schema.QueryAppender
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
	q.subquery = sq
	return sq
}

func (q *AlterTableQuery) RenameColumn(column, to string) *RenameColumnQuery {
	sq := newRenameColumnQuery(q.db, q, column, to)
	q.subquery = sq
	return sq
}

// ------------------------------------------------------------------------------

func (q *AlterTableQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = append(b, "ALTER TABLE "...)

	if sub, ok := q.subquery.(ifExistsAppender); ok {
		b, err = sub.AppendIfExists(fmter, b)
	}

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RENAME ---------------------------------------------------------------------

type RenameTableQuery struct {
	baseQuery
	ifExists
	root    *AlterTableQuery
	newName schema.RenameQueryArg
}

var (
	_ schema.QueryAppender = (*RenameTableQuery)(nil)
	_ ifExistsAppender     = (*RenameTableQuery)(nil)
)

func newRenameTableQuery(db *DB, root *AlterTableQuery, newName string) *RenameTableQuery {
	return &RenameTableQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		root:    root,
		newName: schema.RenameQueryArg{To: newName},
	}
}

func (q *RenameTableQuery) IfExists() *RenameTableQuery {
	q.ifExists = true
	return q
}

func (q *RenameTableQuery) AppendIfExists(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.ifExists.AppendQuery(fmter, b)
}

func (q *RenameTableQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b, err = q.root.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.newName.IsZero() {
		return b, nil
	}

	b = append(b, " RENAME "...)
	b, err = q.newName.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RENAME COLUMN --------------------------------------------------------------

type RenameColumnQuery struct {
	baseQuery
	ifExists
	root    *AlterTableQuery
	newName schema.RenameQueryArg
}

var (
	_ schema.QueryAppender = (*RenameColumnQuery)(nil)
	_ ifExistsAppender     = (*RenameColumnQuery)(nil)
)

func newRenameColumnQuery(db *DB, root *AlterTableQuery, oldName, newName string) *RenameColumnQuery {
	return &RenameColumnQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		root:    root,
		newName: schema.RenameQueryArg{Original: oldName, To: newName},
	}
}

func (q *RenameColumnQuery) IfExists() *RenameColumnQuery {
	q.ifExists = true
	return q
}

func (q *RenameColumnQuery) AppendIfExists(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	return q.ifExists.AppendQuery(fmter, b)
}

func (q *RenameColumnQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	b, err = q.root.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	if q.newName.IsZero() {
		return b, nil
	}

	b = append(b, " RENAME COLUMN "...)
	b, err = q.newName.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
