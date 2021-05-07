package schema

import (
	"context"
	"reflect"
)

type BeforeScanHook interface {
	BeforeScan(context.Context) error
}

var beforeScanHookType = reflect.TypeOf((*BeforeScanHook)(nil)).Elem()

//------------------------------------------------------------------------------

type AfterScanHook interface {
	AfterScan(context.Context) error
}

var afterScanHookType = reflect.TypeOf((*AfterScanHook)(nil)).Elem()

//------------------------------------------------------------------------------

type AfterSelectHook interface {
	AfterSelect(context.Context) error
}

var afterSelectHookType = reflect.TypeOf((*AfterSelectHook)(nil)).Elem()

//------------------------------------------------------------------------------

type BeforeInsertHook interface {
	BeforeInsert(context.Context) error
}

var beforeInsertHookType = reflect.TypeOf((*BeforeInsertHook)(nil)).Elem()

//------------------------------------------------------------------------------

type AfterInsertHook interface {
	AfterInsert(context.Context) error
}

var afterInsertHookType = reflect.TypeOf((*AfterInsertHook)(nil)).Elem()

//------------------------------------------------------------------------------

type BeforeUpdateHook interface {
	BeforeUpdate(context.Context) error
}

var beforeUpdateHookType = reflect.TypeOf((*BeforeUpdateHook)(nil)).Elem()

//------------------------------------------------------------------------------

type AfterUpdateHook interface {
	AfterUpdate(context.Context) error
}

var afterUpdateHookType = reflect.TypeOf((*AfterUpdateHook)(nil)).Elem()

//------------------------------------------------------------------------------

type BeforeDeleteHook interface {
	BeforeDelete(context.Context) error
}

var beforeDeleteHookType = reflect.TypeOf((*BeforeDeleteHook)(nil)).Elem()

//------------------------------------------------------------------------------

type AfterDeleteHook interface {
	AfterDelete(context.Context) error
}

var afterDeleteHookType = reflect.TypeOf((*AfterDeleteHook)(nil)).Elem()
