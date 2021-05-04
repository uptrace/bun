package bun

import (
	"context"
	"database/sql"
	"sort"

	"github.com/uptrace/bun/sqlfmt"
)

type mapModel struct {
	hookStubs

	db *DB

	ptr *map[string]interface{}
	m   map[string]interface{}

	columns   []string
	scanIndex int
}

var _ model = (*mapModel)(nil)

func newMapModel(db *DB, ptr *map[string]interface{}) *mapModel {
	m := &mapModel{
		db:  db,
		ptr: ptr,
	}
	if ptr != nil {
		m.m = *ptr
	}
	return m
}

func (m *mapModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	if !rows.Next() {
		return 0, sql.ErrNoRows
	}

	if err := m.ScanRow(ctx, rows); err != nil {
		return 0, err
	}

	return 1, nil
}

func (m *mapModel) ScanRow(ctx context.Context, rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	m.columns = columns
	dest := makeDest(m, len(columns))

	if m.m == nil {
		m.m = make(map[string]interface{}, len(m.columns))
	}
	if err := rows.Scan(dest...); err != nil {
		return err
	}
	*m.ptr = m.m

	return nil
}

func (m *mapModel) Scan(src interface{}) error {
	columnName := m.columns[m.scanIndex]
	m.scanIndex++
	m.m[columnName] = src
	return nil
}

func (m *mapModel) appendColumnsValues(fmter sqlfmt.QueryFormatter, b []byte) []byte {
	keys := make([]string, 0, len(m.m))

	for k := range m.m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b = append(b, " ("...)

	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = sqlfmt.AppendIdent(fmter, b, k)
	}

	b = append(b, ") VALUES ("...)

	isTemplate := sqlfmt.IsNopFormatter(fmter)
	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}
		if isTemplate {
			b = append(b, '?')
		} else {
			b = sqlfmt.Append(fmter, b, m.m[k])
		}
	}

	b = append(b, ")"...)

	return b
}

func (m *mapModel) appendSet(fmter sqlfmt.QueryFormatter, b []byte) []byte {
	keys := make([]string, 0, len(m.m))

	for k := range m.m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	isTemplate := sqlfmt.IsNopFormatter(fmter)
	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}

		b = sqlfmt.AppendIdent(fmter, b, k)
		b = append(b, " = "...)
		if isTemplate {
			b = append(b, '?')
		} else {
			b = sqlfmt.Append(fmter, b, m.m[k])
		}
	}

	return b
}

func makeDest(v interface{}, n int) []interface{} {
	dest := make([]interface{}, n)
	for i := range dest {
		dest[i] = v
	}
	return dest
}
