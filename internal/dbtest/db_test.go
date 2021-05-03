package dbtest_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	//_ "github.com/go-sql-driver/mysql"
	_ "github.com/go-mysql-org/go-mysql/driver"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

var ctx = context.TODO()

func pg(t *testing.T) *bun.DB {
	pg, err := sql.Open("pgx", "postgres://postgres:@localhost:5432/test?sslmode=disable")
	require.NoError(t, err)

	return bun.Open(pg, pgdialect.New())
}

func sqlite(t *testing.T) *bun.DB {
	sqlite, err := sql.Open("sqlite3", ":memory:?cache=shared")
	require.NoError(t, err)

	sqlite.SetMaxIdleConns(1000)
	sqlite.SetConnMaxLifetime(0)

	return bun.Open(sqlite, sqlitedialect.New())
}

func mysql(t *testing.T) *bun.DB {
	mysql, err := sql.Open("mysql", "root:pass@127.0.0.1:3306?test") //"root:pass@/test")
	require.NoError(t, err)

	return bun.Open(mysql, mysqldialect.New())
}

func dbs(t *testing.T) []*bun.DB {
	return []*bun.DB{
		pg(t),
		sqlite(t),
		mysql(t),
	}
}

func TestSelectScan(t *testing.T) {
	type Test struct {
		name string
		run  func(t *testing.T, db *bun.DB)
	}

	tests := []Test{
		{"testSelectScan", testSelectScan},
		{"testSelectCount", testSelectCount},
		{"testSelectMap", testSelectMap},
		{"testSelectMapSlice", testSelectMapSlice},
		{"testSelectStruct", testSelectStruct},
		{"testSelectNestedStructValue", testSelectNestedStructValue},
		{"testSelectNestedStructPtr", testSelectNestedStructPtr},
		{"testSelectStructSlice", testSelectStructSlice},
		{"testSelectSingleSlice", testSelectSingleSlice},
		{"testSelectMultiSlice", testSelectMultiSlice},
		{"testSelectJSON", testSelectJSON},
		{"testScanRows", testScanRows},
	}

	for _, db := range dbs(t) {
		t.Run(db.Dialect().Name(), func(t *testing.T) {
			defer db.Close()

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					test.run(t, db)
				})
			}
		})
	}
}

func testSelectScan(t *testing.T, db *bun.DB) {
	var num int
	err := db.NewSelect().ColumnExpr("10").Scan(ctx, &num)
	require.NoError(t, err)
	require.Equal(t, 10, num)

	err = db.NewSelect().ColumnExpr("42").Where("FALSE").Scan(ctx, &num)
	require.EqualError(t, err, "sql: no rows in result set")
}

func testSelectCount(t *testing.T, db *bun.DB) {
	values := db.NewValues(&[]map[string]interface{}{
		{"num": 1},
		{"num": 2},
		{"num": 3},
	})

	count, err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		OrderExpr("t.num DESC").
		Limit(1).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func testSelectMap(t *testing.T, db *bun.DB) {
	var m map[string]interface{}
	err := db.NewSelect().
		ColumnExpr("10 AS num").
		Scan(ctx, &m)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"num": int64(10),
	}, m)
}

func testSelectMapSlice(t *testing.T, db *bun.DB) {
	values := db.NewValues(&[]map[string]interface{}{
		{"column1": 1},
		{"column1": 2},
		{"column1": 3},
	})

	var ms []map[string]interface{}
	err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		Scan(ctx, &ms)
	require.NoError(t, err)
	require.Len(t, ms, 3)
	for i, m := range ms {
		require.Equal(t, map[string]interface{}{
			"column1": int64(i + 1),
		}, m)
	}
}

func testSelectStruct(t *testing.T, db *bun.DB) {
	type Model struct {
		Num int
		Str string
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("10 AS num, 'hello' as str").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, 10, model.Num)
	require.Equal(t, "hello", model.Str)

	err = db.NewSelect().ColumnExpr("42").Where("FALSE").Scan(ctx, model)
	require.EqualError(t, err, "sql: no rows in result set")

	err = db.NewSelect().ColumnExpr("1 as unknown_column").Scan(ctx, model)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Model does not have column")
}

func testSelectNestedStructValue(t *testing.T, db *bun.DB) {
	type Model struct {
		Num int
		Sub struct {
			Str string
		}
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("10 AS num, 'hello' as sub__str").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, 10, model.Num)
	require.Equal(t, "hello", model.Sub.Str)
}

func testSelectNestedStructPtr(t *testing.T, db *bun.DB) {
	type Sub struct {
		Str string
	}

	type Model struct {
		Num int
		Sub *Sub
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("10 AS num").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, 10, model.Num)
	require.Nil(t, model.Sub)

	model = new(Model)
	err = db.NewSelect().
		ColumnExpr("10 AS num, 'hello' AS sub__str").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, 10, model.Num)
	require.Equal(t, "hello", model.Sub.Str)

	model = new(Model)
	err = db.NewSelect().
		ColumnExpr("10 AS num, NULL AS sub__str").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, 10, model.Num)
	require.Nil(t, model.Sub)
}

func testSelectStructSlice(t *testing.T, db *bun.DB) {
	type Model struct {
		Num int `bun:"column1"`
	}

	values := db.NewValues(&[]map[string]interface{}{
		{"column1": 1},
		{"column1": 2},
		{"column1": 3},
	})

	models := make([]Model, 0)
	err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		Scan(ctx, &models)
	require.NoError(t, err)
	require.Len(t, models, 3)
	for i, model := range models {
		require.Equal(t, i+1, model.Num)
	}
}

func testSelectSingleSlice(t *testing.T, db *bun.DB) {
	values := db.NewValues(&[]map[string]interface{}{
		{"column1": 1},
		{"column1": 2},
		{"column1": 3},
	})

	var ns []int
	err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		Scan(ctx, &ns)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, ns)
}

func testSelectMultiSlice(t *testing.T, db *bun.DB) {
	values := db.NewValues(&[]map[string]interface{}{
		{"a": 1, "b": "foo"},
		{"a": 2, "b": "bar"},
		{"a": 3, "b": ""},
	})

	var ns []int
	var ss []string
	err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		Scan(ctx, &ns, &ss)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, ns)
	require.Equal(t, []string{"foo", "bar", ""}, ss)
}

func testSelectJSON(t *testing.T, db *bun.DB) {
	type Model struct {
		Map map[string]string
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("? AS map", map[string]string{"hello": "world"}).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"hello": "world"}, model.Map)
}

func testScanRows(t *testing.T, db *bun.DB) {
	rows, err := db.QueryContext(ctx, "SELECT 42")
	require.NoError(t, err)
	defer rows.Close()

	var num int
	err = db.ScanRows(ctx, rows, &num)
	require.NoError(t, err)
	require.Equal(t, 42, num)
}
