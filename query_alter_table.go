package bun

import "github.com/uptrace/bun/schema"

type AlterTableQuery struct {
	baseQuery

	// TODO: collect in an array, use internal.Warn.Printf to warn of multiple conflicting RENAME COLUMN entries
	renameColumn schema.RenameQueryArg
}

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

func (q *AlterTableQuery) RenameColumn(column, to string) *AlterTableQuery {
	q.renameColumn = schema.RenameQueryArg{Original: column, To: to}
	return q
}

// ------------------------------------------------------------------------------

func (q *AlterTableQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = append(b, "ALTER TABLE "...)
	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " RENAME COLUMN "...)
	b, err = q.renameColumn.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
