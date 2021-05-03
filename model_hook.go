package bun

import (
	"context"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type hookStubs struct{}

var (
	_ AfterSelectHook  = (*hookStubs)(nil)
	_ BeforeInsertHook = (*hookStubs)(nil)
	_ AfterInsertHook  = (*hookStubs)(nil)
	_ BeforeUpdateHook = (*hookStubs)(nil)
	_ AfterUpdateHook  = (*hookStubs)(nil)
	_ BeforeDeleteHook = (*hookStubs)(nil)
	_ AfterDeleteHook  = (*hookStubs)(nil)
)

func (hookStubs) AfterSelect(ctx context.Context) error                     { return nil }
func (hookStubs) BeforeInsert(ctx context.Context) (context.Context, error) { return ctx, nil }
func (hookStubs) AfterInsert(ctx context.Context) error                     { return nil }
func (hookStubs) BeforeUpdate(ctx context.Context) (context.Context, error) { return ctx, nil }
func (hookStubs) AfterUpdate(ctx context.Context) error                     { return nil }
func (hookStubs) BeforeDelete(ctx context.Context) (context.Context, error) { return ctx, nil }
func (hookStubs) AfterDelete(ctx context.Context) error                     { return nil }

func callHookSlice(
	ctx context.Context,
	slice reflect.Value,
	ptr bool,
	hook func(context.Context, reflect.Value) (context.Context, error),
) (context.Context, error) {
	var firstErr error
	sliceLen := slice.Len()
	for i := 0; i < sliceLen; i++ {
		v := slice.Index(i)
		if !ptr {
			v = v.Addr()
		}

		var err error
		ctx, err = hook(ctx, v)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return ctx, firstErr
}

func callHookSlice2(
	ctx context.Context,
	slice reflect.Value,
	ptr bool,
	hook func(context.Context, reflect.Value) error,
) error {
	var firstErr error
	if slice.IsValid() {
		sliceLen := slice.Len()
		for i := 0; i < sliceLen; i++ {
			v := slice.Index(i)
			if !ptr {
				v = v.Addr()
			}

			err := hook(ctx, v)
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

//------------------------------------------------------------------------------

func callBeforeScanHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.BeforeScanHook).BeforeScan(ctx)
}

func callAfterScanHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.AfterScanHook).AfterScan(ctx)
}

func callAfterSelectHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.AfterSelectHook).AfterSelect(ctx)
}

func callAfterSelectHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) error {
	return callHookSlice2(ctx, slice, ptr, callAfterSelectHook)
}

func callBeforeInsertHook(ctx context.Context, v reflect.Value) (context.Context, error) {
	return v.Interface().(schema.BeforeInsertHook).BeforeInsert(ctx)
}

func callBeforeInsertHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) (context.Context, error) {
	return callHookSlice(ctx, slice, ptr, callBeforeInsertHook)
}

func callAfterInsertHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.AfterInsertHook).AfterInsert(ctx)
}

func callAfterInsertHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) error {
	return callHookSlice2(ctx, slice, ptr, callAfterInsertHook)
}

func callBeforeUpdateHook(ctx context.Context, v reflect.Value) (context.Context, error) {
	return v.Interface().(schema.BeforeUpdateHook).BeforeUpdate(ctx)
}

func callBeforeUpdateHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) (context.Context, error) {
	return callHookSlice(ctx, slice, ptr, callBeforeUpdateHook)
}

func callAfterUpdateHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.AfterUpdateHook).AfterUpdate(ctx)
}

func callAfterUpdateHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) error {
	return callHookSlice2(ctx, slice, ptr, callAfterUpdateHook)
}

func callBeforeDeleteHook(ctx context.Context, v reflect.Value) (context.Context, error) {
	return v.Interface().(schema.BeforeDeleteHook).BeforeDelete(ctx)
}

func callBeforeDeleteHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) (context.Context, error) {
	return callHookSlice(ctx, slice, ptr, callBeforeDeleteHook)
}

func callAfterDeleteHook(ctx context.Context, v reflect.Value) error {
	return v.Interface().(schema.AfterDeleteHook).AfterDelete(ctx)
}

func callAfterDeleteHookSlice(
	ctx context.Context, slice reflect.Value, ptr bool,
) error {
	return callHookSlice2(ctx, slice, ptr, callAfterDeleteHook)
}
