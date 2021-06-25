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
	for _, db := range dbs(t) {
		t.Run(db.Dialect().Name(), func(t *testing.T) {
			defer db.Close()

			testModelHook(t, db)
		})
	}
}

func testModelHook(t *testing.T, db *bun.DB) {
	_, err := db.NewDropTable().Model((*ModelHookTest)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Model((*ModelHookTest)(nil)).Exec(ctx)
	require.NoError(t, err)

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewInsert().Model(hook).Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeInsert", "AfterInsert"}, events.Flush())
	}

	{
		hook := new(ModelHookTest)
		err := db.NewSelect().Model(hook).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{
			"BeforeSelect",
			"BeforeScan",
			"AfterScan",
			"AfterSelect",
		}, events.Flush())
	}

	{
		hooks := make([]ModelHookTest, 0)
		err := db.NewSelect().Model(&hooks).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{
			"BeforeSelect",
			"BeforeScan",
			"AfterScan",
			"AfterSelect",
		}, events.Flush())
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewUpdate().Model(hook).Where("id = 1").Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeUpdate", "AfterUpdate"}, events.Flush())
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewDelete().Model(hook).Where("id = 1").Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeDelete", "AfterDelete"}, events.Flush())
	}

	{
		_, err := db.NewDelete().Model((*ModelHookTest)(nil)).Where("TRUE").Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeDelete", "AfterDelete"}, events.Flush())
	}
}

type ModelHookTest struct {
	ID    int
	Value string
}

var _ bun.BeforeScanHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeScan(c context.Context) error {
	events.Add("BeforeScan")
	return nil
}

var _ bun.AfterScanHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterScan(c context.Context) error {
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
	case *ModelHookTest, *[]ModelHookTest:
		// ok
	default:
		panic(fmt.Errorf("unexpected: %T", value))
	}
}
