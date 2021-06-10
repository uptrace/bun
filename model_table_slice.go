package bun

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type sliceTableModel struct {
	structTableModel

	slice      reflect.Value
	sliceLen   int
	sliceOfPtr bool
	nextElem   func() reflect.Value
}

var _ tableModel = (*sliceTableModel)(nil)

func newSliceTableModel(
	db *DB, dest interface{}, slice reflect.Value, elemType reflect.Type,
) *sliceTableModel {
	m := &sliceTableModel{
		structTableModel: structTableModel{
			db:    db,
			table: db.Table(elemType),
			dest:  dest,
			root:  slice,
		},

		slice:    slice,
		sliceLen: slice.Len(),
		nextElem: makeSliceNextElemFunc(slice),
	}
	m.init(slice.Type())
	return m
}

func (m *sliceTableModel) init(sliceType reflect.Type) {
	switch sliceType.Elem().Kind() {
	case reflect.Ptr, reflect.Interface:
		m.sliceOfPtr = true
	}
}

func (m *sliceTableModel) Join(name string, apply func(*SelectQuery) *SelectQuery) *join {
	return m.join(m.slice, name, apply)
}

func (m *sliceTableModel) Bind(bind reflect.Value) {
	m.slice = bind.Field(m.index[len(m.index)-1])
}

func (m *sliceTableModel) SetCap(cap int) {
	if cap > 100 {
		cap = 100
	}
	if m.slice.Cap() < cap {
		m.slice.Set(reflect.MakeSlice(m.slice.Type(), 0, cap))
	}
}

func (m *sliceTableModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	m.columns = columns
	dest := makeDest(m, len(columns))

	if m.slice.IsValid() && m.slice.Len() > 0 {
		m.slice.Set(m.slice.Slice(0, 0))
	}

	var n int

	for rows.Next() {
		m.strct = m.nextElem()
		m.structInited = false

		if err := m.scanRow(ctx, rows, dest); err != nil {
			return 0, err
		}

		n++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	return n, nil
}

// Inherit these hooks from structTableModel.
var (
	_ schema.BeforeScanHook = (*sliceTableModel)(nil)
	_ schema.AfterScanHook  = (*sliceTableModel)(nil)
)

func (m *sliceTableModel) updateSoftDeleteField() error {
	sliceLen := m.slice.Len()
	for i := 0; i < sliceLen; i++ {
		strct := indirect(m.slice.Index(i))
		fv := m.table.SoftDeleteField.Value(strct)
		if err := m.table.UpdateSoftDeleteField(fv); err != nil {
			return err
		}
	}
	return nil
}
