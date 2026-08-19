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

	t.Run("count", func(t *testing.T) {
		_, err := db.NewSelect().Model((*ModelHookTest)(nil)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeSelect"}, events.Flush())
	})

	t.Run("exists", func(t *testing.T) {
		_, err := db.NewSelect().Model((*ModelHookTest)(nil)).Exists(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeSelect"}, events.Flush())
	})

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

// TenantFilterModel uses BeforeSelect to add a WHERE clause, simulating
// multi-tenant row filtering. This verifies that Count and Exists honour
// the hook — the bug that motivated this test is that they previously
// skipped BeforeSelect entirely (see https://github.com/uptrace/bun/issues/1289).
type TenantFilterModel struct {
	bun.BaseModel `bun:"table:tenant_filter_models"`

	ID       int64  `bun:",pk,autoincrement"`
	TenantID int64
	Value    string
}

// tenantKey is a context key for the tenant ID used in BeforeSelect.
type tenantKey struct{}

var _ bun.BeforeSelectHook = (*TenantFilterModel)(nil)

func (m *TenantFilterModel) BeforeSelect(ctx context.Context, query *bun.SelectQuery) error {
	if tid, ok := ctx.Value(tenantKey{}).(int64); ok {
		query.Where("tenant_id = ?", tid)
	}
	return nil
}

func TestBeforeSelectHookOnCountAndExists(t *testing.T) {
	testEachDB(t, testBeforeSelectHookOnCountAndExists)
}

func testBeforeSelectHookOnCountAndExists(t *testing.T, dbName string, db *bun.DB) {
	mustResetModel(t, ctx, db, (*TenantFilterModel)(nil))

	// Seed two tenants with different row counts.
	models := []TenantFilterModel{
		{TenantID: 1, Value: "a"},
		{TenantID: 1, Value: "b"},
		{TenantID: 1, Value: "c"},
		{TenantID: 2, Value: "d"},
	}
	_, err := db.NewInsert().Model(&models).Exec(ctx)
	require.NoError(t, err)

	t.Run("count_with_tenant_filter", func(t *testing.T) {
		tenant1Ctx := context.WithValue(ctx, tenantKey{}, int64(1))
		count, err := db.NewSelect().Model((*TenantFilterModel)(nil)).Count(tenant1Ctx)
		require.NoError(t, err)
		require.Equal(t, int64(3), count, "Count must return only tenant 1's rows")

		tenant2Ctx := context.WithValue(ctx, tenantKey{}, int64(2))
		count, err = db.NewSelect().Model((*TenantFilterModel)(nil)).Count(tenant2Ctx)
		require.NoError(t, err)
		require.Equal(t, int64(1), count, "Count must return only tenant 2's rows")
	})

	t.Run("count_without_tenant_filter", func(t *testing.T) {
		// No tenant in context — hook adds no WHERE, so all rows are counted.
		count, err := db.NewSelect().Model((*TenantFilterModel)(nil)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(4), count)
	})

	t.Run("exists_with_tenant_filter", func(t *testing.T) {
		tenant1Ctx := context.WithValue(ctx, tenantKey{}, int64(1))
		exists, err := db.NewSelect().Model((*TenantFilterModel)(nil)).Exists(tenant1Ctx)
		require.NoError(t, err)
		require.True(t, exists, "Exists must find tenant 1's rows")

		// Tenant 99 has no rows.
		tenant99Ctx := context.WithValue(ctx, tenantKey{}, int64(99))
		exists, err = db.NewSelect().Model((*TenantFilterModel)(nil)).Exists(tenant99Ctx)
		require.NoError(t, err)
		require.False(t, exists, "Exists must not find rows for a tenant with none")
	})
}

func assertQueryModel(query interface{ GetModel() bun.Model }) {
	switch value := query.GetModel().Value(); value.(type) {
	case *ModelHookTest, *[]ModelHookTest, *[]*ModelHookTest:
		// ok
	default:
		panic(fmt.Errorf("unexpected: %T", value))
	}
}
