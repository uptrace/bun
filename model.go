package bun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/uptrace/bun/schema"
)

var errModelNil = errors.New("bun: Model(nil)")

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

type hooklessModel interface {
	ScanRows(ctx context.Context, rows *sql.Rows) (int, error)
}

type model interface {
	hooklessModel

	schema.AfterSelectHook

	schema.BeforeInsertHook
	schema.AfterInsertHook

	schema.BeforeUpdateHook
	schema.AfterUpdateHook

	schema.BeforeDeleteHook
	schema.AfterDeleteHook
}

type tableModel interface {
	model

	schema.BeforeScanHook
	schema.AfterScanHook
	ScanColumn(column string, src interface{}) error

	IsNil() bool
	Table() *schema.Table
	Relation() *schema.Relation

	Join(string, func(*SelectQuery) *SelectQuery) *join
	GetJoin(string) *join
	GetJoins() []join
	AddJoin(join) *join

	Root() reflect.Value
	Index() []int
	ParentIndex() []int
	Mount(reflect.Value)
	Kind() reflect.Kind
	Value() reflect.Value

	updateSoftDeleteField() error
}

func newModel(db *DB, dest []interface{}) (model, error) {
	if len(dest) == 1 {
		return _newModel(db, dest[0], true)
	}

	values := make([]reflect.Value, len(dest))

	for i, el := range dest {
		v := reflect.ValueOf(el)
		if v.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("bun: Scan(non-pointer %T)", dest)
		}

		v = v.Elem()
		if v.Kind() != reflect.Slice {
			return newScanModel(dest), nil
		}

		values[i] = v
	}

	return newSliceModel(db, values), nil
}

func newSingleModel(db *DB, dest interface{}) (model, error) {
	return _newModel(db, dest, false)
}

func _newModel(db *DB, dest interface{}, scan bool) (model, error) {
	switch dest := dest.(type) {
	case nil:
		return nil, errModelNil
	case model:
		return dest, nil
	case hooklessModel:
		return newModelWithHookStubs(dest), nil
	case sql.Scanner:
		if !scan {
			return nil, fmt.Errorf("bun: Model(unsupported %T)", dest)
		}
		return newScanModel([]interface{}{dest}), nil
	}

	v := reflect.ValueOf(dest)
	if !v.IsValid() {
		return nil, errModelNil
	}
	if v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("bun: Model(non-pointer %T)", dest)
	}

	if v.IsNil() {
		typ := v.Type().Elem()
		if typ.Kind() == reflect.Struct {
			return newStructTableModel(db, db.Table(typ)), nil
		}
		return nil, fmt.Errorf("bun: Model(nil %T)", dest)
	}

	v = v.Elem()

	switch v.Kind() {
	case reflect.Map:
		typ := v.Type()
		if err := validMap(typ); err != nil {
			return nil, err
		}
		mapPtr := v.Addr().Interface().(*map[string]interface{})
		return newMapModel(db, mapPtr), nil
	case reflect.Struct:
		if v.Type() != timeType {
			return newStructTableModelValue(db, v), nil
		}
	case reflect.Slice:
		switch elemType := sliceElemType(v); elemType.Kind() {
		case reflect.Struct:
			if elemType != timeType {
				return newSliceTableModel(db, v, elemType), nil
			}
		case reflect.Map:
			if err := validMap(elemType); err != nil {
				return nil, err
			}
			slicePtr := v.Addr().Interface().(*[]map[string]interface{})
			return newMapSliceModel(db, slicePtr), nil
		}
		return newSliceModel(db, []reflect.Value{v}), nil
	}

	if scan {
		return newScanModel([]interface{}{dest}), nil
	}

	return nil, fmt.Errorf("bun: Model(unsupported %T)", dest)
}

func newTableModelIndex(
	db *DB,
	table *schema.Table,
	root reflect.Value,
	index []int,
	rel *schema.Relation,
) (tableModel, error) {
	typ := typeByIndex(table.Type, index)

	if typ.Kind() == reflect.Struct {
		return &structTableModel{
			db:    db,
			table: table.Dialect().Tables().Get(typ),
			rel:   rel,

			root:  root,
			index: index,
		}, nil
	}

	if typ.Kind() == reflect.Slice {
		structType := indirectType(typ.Elem())
		if structType.Kind() == reflect.Struct {
			m := sliceTableModel{
				structTableModel: structTableModel{
					db:    db,
					table: table.Dialect().Tables().Get(structType),
					rel:   rel,

					root:  root,
					index: index,
				},
			}
			m.init(typ)
			return &m, nil
		}
	}

	return nil, fmt.Errorf("bun: NewModel(%s)", typ)
}

func validMap(typ reflect.Type) error {
	if typ.Key().Kind() != reflect.String || typ.Elem().Kind() != reflect.Interface {
		return fmt.Errorf("bun: Model(unsupported %s) (expected *map[string]interface{})",
			typ)
	}
	return nil
}

//------------------------------------------------------------------------------

type modelWithHookStubs struct {
	hookStubs
	hooklessModel
}

func newModelWithHookStubs(m hooklessModel) model {
	return modelWithHookStubs{
		hooklessModel: m,
	}
}
