package bun

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type sliceInfo struct {
	nextElem func() reflect.Value
	scan     schema.ScannerFunc
}

type sliceModel struct {
	hookStubs

	dest      []reflect.Value
	scanIndex int
	info      []sliceInfo
}

var _ model = (*sliceModel)(nil)

func newSliceModel(db *DB, dest []reflect.Value) *sliceModel {
	return &sliceModel{
		dest: dest,
	}
}

func (m *sliceModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	m.info = make([]sliceInfo, len(m.dest))
	for i, v := range m.dest {
		if v.IsValid() && v.Len() > 0 {
			v.Set(v.Slice(0, 0))
		}

		m.info[i] = sliceInfo{
			nextElem: internal.MakeSliceNextElemFunc(v),
			scan:     schema.Scanner(v.Type().Elem()),
		}
	}

	if len(columns) == 0 {
		return 0, nil
	}
	dest := makeDest(m, len(columns))

	var n int

	for rows.Next() {
		m.scanIndex = 0
		if err := rows.Scan(dest...); err != nil {
			return 0, err
		}
		n++
	}

	return n, nil
}

func (m *sliceModel) Scan(src interface{}) error {
	info := m.info[m.scanIndex]
	m.scanIndex++

	dest := info.nextElem()
	return info.scan(dest, src)
}
