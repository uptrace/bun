package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

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
		require.Equal(t, []string{"BeforeInsert", "AfterInsert"}, hook.events)
	}

	{
		hook := new(ModelHookTest)
		err := db.NewSelect().Model(hook).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeScan", "AfterScan", "AfterSelect"}, hook.events)
	}

	{
		hooks := make([]ModelHookTest, 0)
		err := db.NewSelect().Model(&hooks).Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeScan", "AfterScan", "AfterSelect"}, hooks[0].events)
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewUpdate().Model(hook).WherePK().Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeUpdate", "AfterUpdate"}, hook.events)
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewDelete().Model(hook).WherePK().Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeDelete", "AfterDelete"}, hook.events)
	}

	{
		_, err := db.NewDelete().Model((*ModelHookTest)(nil)).Where("TRUE").Exec(ctx)
		require.NoError(t, err)
	}
}

type ModelHookTest struct {
	ID    int
	Value string

	events []string
}

var _ bun.BeforeScanHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeScan(c context.Context) error {
	t.events = append(t.events, "BeforeScan")
	return nil
}

var _ bun.AfterScanHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterScan(c context.Context) error {
	t.events = append(t.events, "AfterScan")
	return nil
}

var _ bun.AfterSelectHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterSelect(c context.Context) error {
	t.events = append(t.events, "AfterSelect")
	return nil
}

var _ bun.BeforeInsertHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeInsert(c context.Context) (context.Context, error) {
	t.events = append(t.events, "BeforeInsert")
	return c, nil
}

var _ bun.AfterInsertHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterInsert(c context.Context) error {
	t.events = append(t.events, "AfterInsert")
	return nil
}

var _ bun.BeforeUpdateHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeUpdate(c context.Context) (context.Context, error) {
	t.events = append(t.events, "BeforeUpdate")
	return c, nil
}

var _ bun.AfterUpdateHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterUpdate(c context.Context) error {
	t.events = append(t.events, "AfterUpdate")
	return nil
}

var _ bun.BeforeDeleteHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeDelete(c context.Context) (context.Context, error) {
	t.events = append(t.events, "BeforeDelete")
	return c, nil
}

var _ bun.AfterDeleteHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterDelete(c context.Context) error {
	t.events = append(t.events, "AfterDelete")
	return nil
}
