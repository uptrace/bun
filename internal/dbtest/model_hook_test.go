package dbtest_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

var events Events

type Events struct {
	mu sync.Mutex
	ss []string
}

func (es *Events) Add(event string) {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.ss = append(es.ss, event)
}

func (es *Events) Flush() []string {
	es.mu.Lock()
	defer es.mu.Unlock()

	ss := es.ss
	es.ss = nil
	return ss
}

func TestModelHook(t *testing.T) {
	testEachDB(t, testModelHook)
}

func testModelHook(t *testing.T, dbName string, db *bun.DB) {
	mustResetModel(t, ctx, db, (*ModelHookTest)(nil))

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewInsert().Model(hook).Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeInsert", "BeforeAppendModel", "AfterInsert"}, events.Flush())
	}

	{
		hook := new(ModelHookTest)
		err := db.NewSelect().Model(hook).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{
			"BeforeSelect",
			"BeforeAppendModel",
			"BeforeScan",
			"AfterScan",
			"AfterSelect",
		}, events.Flush())
	}

	t.Run("selectEmptySlice", func(t *testing.T) {
		hooks := make([]ModelHookTest, 0)
		err := db.NewSelect().Model(&hooks).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{
			"BeforeSelect",
			"BeforeScan",
			"AfterScan",
			"AfterSelect",
		}, events.Flush())
	})

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewUpdate().Model(hook).Where("id = 1").Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeUpdate", "BeforeAppendModel", "AfterUpdate"}, events.Flush())
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewDelete().Model(hook).Where("id = 1").Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeDelete", "BeforeAppendModel", "AfterDelete"}, events.Flush())
	}

	{
		_, err := db.NewDelete().Model((*ModelHookTest)(nil)).Where("1 = 1").Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeDelete", "AfterDelete"}, events.Flush())
	}

	t.Run("insertSlice", func(t *testing.T) {
		hooks := []ModelHookTest{{ID: 1}, {ID: 2}}
		_, err := db.NewInsert().Model(&hooks).Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{
			"BeforeInsert",
			"BeforeAppendModel",
			"BeforeAppendModel",
			"AfterInsert",
		}, events.Flush())
	})

	t.Run("insertSliceOfPtr", func(t *testing.T) {
		hooks := []*ModelHookTest{{ID: 3}, {ID: 4}}
		_, err := db.NewInsert().Model(&hooks).Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{
			"BeforeInsert",
			"BeforeAppendModel",
			"BeforeAppendModel",
			"AfterInsert",
		}, events.Flush())
	})
}

type ModelHookTest struct {
	ID    int `bun:",pk"`
	Value string
}

var _ bun.BeforeAppendModelHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	events.Add("BeforeAppendModel")
	return nil
}

var _ bun.BeforeScanRowHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeScanRow(ctx context.Context) error {
	events.Add("BeforeScan")
	return nil
}

var _ bun.AfterScanRowHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterScanRow(ctx context.Context) error {
	events.Add("AfterScan")
	return nil
}

var _ bun.BeforeSelectHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeSelect(ctx context.Context, query *bun.SelectQuery) error {
	assertQueryModel(query)
	events.Add("BeforeSelect")
	return nil
}

var _ bun.AfterSelectHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterSelect(ctx context.Context, query *bun.SelectQuery) error {
	assertQueryModel(query)
	events.Add("AfterSelect")
	return nil
}

var _ bun.BeforeUpdateHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeUpdate(ctx context.Context, query *bun.UpdateQuery) error {
	assertQueryModel(query)
	events.Add("BeforeUpdate")
	return nil
}

var _ bun.AfterUpdateHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterUpdate(ctx context.Context, query *bun.UpdateQuery) error {
	assertQueryModel(query)
	events.Add("AfterUpdate")
	return nil
}

var _ bun.BeforeInsertHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeInsert(ctx context.Context, query *bun.InsertQuery) error {
	assertQueryModel(query)
	events.Add("BeforeInsert")
	return nil
}

var _ bun.AfterInsertHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterInsert(ctx context.Context, query *bun.InsertQuery) error {
	assertQueryModel(query)
	events.Add("AfterInsert")
	return nil
}

var _ bun.BeforeDeleteHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeDelete(ctx context.Context, query *bun.DeleteQuery) error {
	assertQueryModel(query)
	events.Add("BeforeDelete")
	return nil
}

var _ bun.AfterDeleteHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterDelete(ctx context.Context, query *bun.DeleteQuery) error {
	assertQueryModel(query)
	events.Add("AfterDelete")
	return nil
}

func assertQueryModel(query interface{ GetModel() bun.Model }) {
	switch value := query.GetModel().Value(); value.(type) {
	case *ModelHookTest, *[]ModelHookTest, *[]*ModelHookTest:
		// ok
	default:
		panic(fmt.Errorf("unexpected: %T", value))
	}
}
