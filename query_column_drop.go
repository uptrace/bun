package bun

import (
	"context"
	"fmt"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type DropColumnQuery struct {
	baseQuery
}

func NewDropColumnQuery(db *DB) *DropColumnQuery {
	q := &DropColumnQuery{
		baseQuery: baseQuery{
			db:  db,
			dbi: db.DB,
		},
	}
	return q
}

func (q *DropColumnQuery) DB(db DBI) *DropColumnQuery {
	q.dbi = db
	return q
}

func (q *DropColumnQuery) Model(model interface{}) *DropColumnQuery {
	q.setTableModel(model)
	return q
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Table(tables ...string) *DropColumnQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *DropColumnQuery) TableExpr(query string, args ...interface{}) *DropColumnQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *DropColumnQuery) ModelTableExpr(query string, args ...interface{}) *DropColumnQuery {
	q.modelTable = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Column(columns ...string) *DropColumnQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

func (q *DropColumnQuery) ColumnExpr(query string, args ...interface{}) *DropColumnQuery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) AppendQuery(fmter schema.Formatter, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}
	if len(q.columns) != 1 {
		return nil, fmt.Errorf("bun: DropColumnQuery requires exactly one column")
	}

	b = append(b, "ALTER TABLE "...)

	b, err = q.appendFirstTable(fmter, b)
	if err != nil {
		return nil, err
	}

	b = append(b, " DROP COLUMN "...)

	b, err = q.columns[0].AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//------------------------------------------------------------------------------

func (q *DropColumnQuery) Exec(ctx context.Context, dest ...interface{}) (res Result, err error) {
	bs := getByteSlice()
	defer putByteSlice(bs)

	queryBytes, err := q.AppendQuery(q.db.fmter, bs.b)
	if err != nil {
		return res, err
	}

	bs.b = queryBytes
	query := internal.String(queryBytes)

	res, err = q.exec(ctx, q, query)
	if err != nil {
		return res, err
	}

	return res, nil
}
