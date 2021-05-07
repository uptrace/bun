package dbtest_test

import (
	"context"
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
			"BeforeScan",
			"AfterScan",
			"AfterSelect",
		}, events.Flush())
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewUpdate().Model(hook).WherePK().Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeUpdate", "AfterUpdate"}, events.Flush())
	}

	{
		hook := &ModelHookTest{ID: 1}
		_, err := db.NewDelete().Model(hook).WherePK().Exec(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"BeforeDelete", "AfterDelete"}, events.Flush())
	}

	{
		_, err := db.NewDelete().Model((*ModelHookTest)(nil)).Where("TRUE").Exec(ctx)
		require.NoError(t, err)
		require.Nil(t, events.Flush())
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

// var _ bun.BeforeSelectQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) BeforeSelectQuery(ctx context.Context, query *bun.SelectQuery) error {
// 	events.Add("BeforeSelectQuery")
// 	return nil
// }

// var _ bun.AfterSelectQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) AfterSelectQuery(ctx context.Context, query *bun.SelectQuery) error {
// 	events.Add("AfterSelectQuery")
// 	return nil
// }

// var _ bun.BeforeUpdateQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) BeforeUpdateQuery(ctx context.Context, query *bun.UpdateQuery) error {
// 	events.Add("BeforeUpdateQuery")
// 	return nil
// }

// var _ bun.AfterUpdateQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) AfterUpdateQuery(ctx context.Context, query *bun.UpdateQuery) error {
// 	events.Add("AfterUpdateQuery")
// 	return nil
// }

// var _ bun.BeforeInsertQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) BeforeInsertQuery(ctx context.Context, query *bun.InsertQuery) error {
// 	events.Add("BeforeInsertQuery")
// 	return nil
// }

// var _ bun.AfterInsertQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) AfterInsertQuery(ctx context.Context, query *bun.InsertQuery) error {
// 	events.Add("AfterInsertQuery")
// 	return nil
// }

// var _ bun.BeforeDeleteQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) BeforeDeleteQuery(ctx context.Context, query *bun.DeleteQuery) error {
// 	events.Add("BeforeDeleteQuery")
// 	return nil
// }

// var _ bun.AfterDeleteQueryHook = (*ModelHookTest)(nil)

// func (t *ModelHookTest) AfterDeleteQuery(ctx context.Context, query *bun.DeleteQuery) error {
// 	events.Add("AfterDeleteQuery")
// 	return nil
// }

var _ bun.AfterSelectHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterSelect(c context.Context) error {
	events.Add("AfterSelect")
	return nil
}

var _ bun.BeforeInsertHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeInsert(ctx context.Context) error {
	events.Add("BeforeInsert")
	return nil
}

var _ bun.AfterInsertHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterInsert(c context.Context) error {
	events.Add("AfterInsert")
	return nil
}

var _ bun.BeforeUpdateHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeUpdate(ctx context.Context) error {
	events.Add("BeforeUpdate")
	return nil
}

var _ bun.AfterUpdateHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterUpdate(c context.Context) error {
	events.Add("AfterUpdate")
	return nil
}

var _ bun.BeforeDeleteHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) BeforeDelete(ctx context.Context) error {
	events.Add("BeforeDelete")
	return nil
}

var _ bun.AfterDeleteHook = (*ModelHookTest)(nil)

func (t *ModelHookTest) AfterDelete(c context.Context) error {
	events.Add("AfterDelete")
	return nil
}
