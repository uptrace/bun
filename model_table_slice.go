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
	db *DB, slice reflect.Value, elemType reflect.Type,
) *sliceTableModel {
	m := &sliceTableModel{
		structTableModel: structTableModel{
			db:    db,
			table: db.Table(elemType),
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

func (m *sliceTableModel) IsNil() bool {
	return false
}

func (m *sliceTableModel) Join(name string, apply func(*SelectQuery) *SelectQuery) *join {
	return m.join(m.Value(), name, apply)
}

func (m *sliceTableModel) Bind(bind reflect.Value) {
	m.slice = bind.Field(m.index[len(m.index)-1])
}

func (m *sliceTableModel) Kind() reflect.Kind {
	return reflect.Slice
}

func (m *sliceTableModel) Value() reflect.Value {
	return m.slice
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

	return n, nil
}

// Inherit these hooks from structTableModel.
var (
	_ schema.BeforeScanHook = (*sliceTableModel)(nil)
	_ schema.AfterScanHook  = (*sliceTableModel)(nil)
)

func (m *sliceTableModel) AfterSelect(ctx context.Context) error {
	if m.table.HasAfterSelectHook() {
		return callAfterSelectHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

func (m *sliceTableModel) BeforeInsert(ctx context.Context) error {
	if m.table.HasBeforeInsertHook() {
		return callBeforeInsertHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

func (m *sliceTableModel) AfterInsert(ctx context.Context) error {
	if m.table.HasAfterInsertHook() {
		return callAfterInsertHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

func (m *sliceTableModel) BeforeUpdate(ctx context.Context) error {
	if m.table.HasBeforeUpdateHook() && !m.IsNil() {
		return callBeforeUpdateHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

func (m *sliceTableModel) AfterUpdate(ctx context.Context) error {
	if m.table.HasAfterUpdateHook() {
		return callAfterUpdateHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

func (m *sliceTableModel) BeforeDelete(ctx context.Context) error {
	if m.table.HasBeforeDeleteHook() && !m.IsNil() {
		return callBeforeDeleteHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

func (m *sliceTableModel) AfterDelete(ctx context.Context) error {
	if m.table.HasAfterDeleteHook() && !m.IsNil() {
		return callAfterDeleteHookSlice(ctx, m.slice, m.sliceOfPtr)
	}
	return nil
}

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
