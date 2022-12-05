package bun

import "github.com/uptrace/bun/schema"

type AlterTableQuery struct {
	baseQuery

	ifExists    bool
	renameTable schema.RenameQueryArg

	// TODO: collect in an array, use internal.Warn.Printf to warn of multiple conflicting RENAME COLUMN entries
	renameColumn schema.RenameQueryArg
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

func (q *AlterTableQuery) Rename(to string) *AlterTableQuery {
	q.renameTable = schema.RenameQueryArg{To: to}
	return q
}

func (q *AlterTableQuery) RenameColumn(column, to string) *AlterTableQuery {
	q.renameColumn = schema.RenameQueryArg{Original: column, To: to}
	return q
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

	if q.ifExists {
		b = append(b, "IF EXISTS "...)
	}

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	if !q.renameTable.IsZero() {
		b = append(b, " RENAME "...)
		b, err = q.renameTable.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	if !q.renameColumn.IsZero() {
		b = append(b, " RENAME COLUMN "...)
		b, err = q.renameColumn.AppendQuery(fmter, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}
