package dbtest_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/mssqldialect"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/extra/bunexp"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/require"
)

var ctx = context.TODO()

const (
	pgName        = "pg"
	pgxName       = "pgx"
	mysql5Name    = "mysql5"
	mysql8Name    = "mysql8"
	mariadbName   = "mariadb"
	sqliteName    = "sqlite"
	mssql2019Name = "mssql2019"
)

var allDBs = map[string]func(tb testing.TB) *bun.DB{
	pgName:        pg,
	pgxName:       pgx,
	mysql5Name:    mysql5,
	mysql8Name:    mysql8,
	mariadbName:   mariadb,
	sqliteName:    sqlite,
	mssql2019Name: mssql2019,
}

var allDialects = []func() schema.Dialect{
	func() schema.Dialect { return pgdialect.New() },
	func() schema.Dialect { return mysqldialect.New() },
	func() schema.Dialect { return sqlitedialect.New() },
	func() schema.Dialect { return mssqldialect.New() },
}

func pg(tb testing.TB) *bun.DB {
	dsn := os.Getenv("PG")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/test?sslmode=disable"
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=pg>", db.String())

	return db
}

func pgx(tb testing.TB) *bun.DB {
	dsn := os.Getenv("PG")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/test?sslmode=disable"
	}

	sqldb, err := sql.Open("pgx", dsn)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=pg>", db.String())

	return db
}

func mysql8(tb testing.TB) *bun.DB {
	dsn := os.Getenv("MYSQL")
	if dsn == "" {
		dsn = "user:pass@/test"
	}

	sqldb, err := sql.Open("mysql", dsn)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, mysqldialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=mysql>", db.String())

	return db
}

func mysql5(tb testing.TB) *bun.DB {
	dsn := os.Getenv("MYSQL5")
	if dsn == "" {
		dsn = "user:pass@tcp(localhost:53306)/test"
	}

	sqldb, err := sql.Open("mysql", dsn)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, mysqldialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=mysql>", db.String())

	return db
}

func mariadb(tb testing.TB) *bun.DB {
	dsn := os.Getenv("MARIADB")
	if dsn == "" {
		dsn = "user:pass@tcp(localhost:13306)/test"
	}

	sqldb, err := sql.Open("mysql", dsn)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, mysqldialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=mysql>", db.String())

	return db
}

func sqlite(tb testing.TB) *bun.DB {
	sqldb, err := sql.Open(sqliteshim.DriverName(), filepath.Join(tb.TempDir(), "sqlite.db"))
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=sqlite>", db.String())

	return db
}

func mssql2019(tb testing.TB) *bun.DB {
	dsn := os.Getenv("MSSQL2019")
	if dsn == "" {
		dsn = "sqlserver://sa:passWORD1@localhost:14339?database=test"
	}

	sqldb, err := sql.Open("sqlserver", dsn)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, sqldb.Close())
	})

	db := bun.NewDB(sqldb, mssqldialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	require.Equal(tb, "DB<dialect=mssql>", db.String())

	return db
}

func testEachDB(t *testing.T, f func(t *testing.T, dbName string, db *bun.DB)) {
	for dbName, newDB := range allDBs {
		t.Run(dbName, func(t *testing.T) {
			f(t, dbName, newDB(t))
		})
	}
}

// testEachDialect allows testing dialect-specific functionality that does not require database interactions.
func testEachDialect(t *testing.T, f func(t *testing.T, dialectName string, dialect schema.Dialect)) {
	for _, newDialect := range allDialects {
		d := newDialect()
		name := d.Name().String()
		t.Run(name, func(t *testing.T) {
			f(t, name, d)
		})
	}
}

func funcName(x interface{}) string {
	s := runtime.FuncForPC(reflect.ValueOf(x).Pointer()).Name()
	if i := strings.LastIndexByte(s, '.'); i >= 0 {
		return s[i+1:]
	}
	return s
}

func TestDB(t *testing.T) {
	type Test struct {
		run func(t *testing.T, db *bun.DB)
	}

	tests := []Test{
		{testPing},
		{testNilModel},
		{testSelectScan},
		{testSelectCount},
		{testSelectMap},
		{testSelectMapSlice},
		{testSelectStruct},
		{testSelectNestedStructValue},
		{testSelectNestedStructPtr},
		{testSelectStructSlice},
		{testSelectSingleSlice},
		{testSelectMultiSlice},
		{testSelectJSONMap},
		{testSelectJSONStruct},
		{testJSONSpecialChars},
		{testSelectRawMessage},
		{testScanNullVar},
		{testScanSingleRow},
		{testScanSingleRowByRow},
		{testScanRows},
		{testRunInTx},
		{testJSONInterface},
		{testJSONValuer},
		{testSelectBool},
		{testRawQuery},
		{testFKViolation},
		{testWithForeignKeysAndRules},
		{testWithForeignKeys},
		{testWithForeignKeysHasMany},
		{testWithPointerForeignKeysHasMany},
		{testWithPointerForeignKeysHasManyWithDriverValuer},
		{testInterfaceAny},
		{testInterfaceJSON},
		{testScanRawMessage},
		{testPointers},
		{testExists},
		{testScanTimeIntoString},
		{testModelNonPointer},
		{testBinaryData},
		{testUpsert},
		{testMultiUpdate},
		{testUpdateWithSkipupdateTag},
		{testScanAndCount},
		{testEmbedModelValue},
		{testEmbedModelPointer},
		{testJSONMarshaler},
		{testNilDriverValue},
		{testRunInTxAndSavepoint},
		{testDriverValuerReturnsItself},
		{testNoPanicWhenReturningNullColumns},
		{testNoForeignKeyForPrimaryKey},
		{testWithPointerPrimaryKeyHasManyWithDriverValuer},
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for _, test := range tests {
			t.Run(funcName(test.run), func(t *testing.T) {
				test.run(t, db)
			})
		}
	})
}

func testPing(t *testing.T, db *bun.DB) {
	err := db.PingContext(ctx)
	require.NoError(t, err)
}

func testNilModel(t *testing.T, db *bun.DB) {
	err := db.NewSelect().ColumnExpr("1").Scan(ctx)
	require.Error(t, err)
	require.Equal(t, "bun: Model(nil)", err.Error())
}

func testSelectScan(t *testing.T, db *bun.DB) {
	var num int
	err := db.NewSelect().ColumnExpr("10").Scan(ctx, &num)
	require.NoError(t, err)
	require.Equal(t, 10, num)

	err = db.NewSelect().
		ColumnExpr("t.num").
		TableExpr("(SELECT 10 AS num) AS t").
		Where("1 = 2").
		Scan(ctx, &num)
	require.Equal(t, sql.ErrNoRows, err)

	var str string
	err = db.NewSelect().ColumnExpr("?", "\\\"'hello\n%_").Scan(ctx, &str)
	require.NoError(t, err)
	require.Equal(t, "\\\"'hello\n%_", str)
}

func testSelectCount(t *testing.T, db *bun.DB) {
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
		return
	}

	values := db.NewValues(&[]map[string]interface{}{
		{"num": 1},
		{"num": 2},
		{"num": 3},
	})

	q := db.NewSelect().
		With("t", values).
		Column("t.num").
		TableExpr("t").
		OrderExpr("t.num DESC").
		Limit(1)

	var num int
	err := q.Scan(ctx, &num)
	require.NoError(t, err)
	require.Equal(t, 3, num)

	count, err := q.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func testSelectMap(t *testing.T, db *bun.DB) {
	var m map[string]interface{}
	err := db.NewSelect().
		ColumnExpr("10 AS num").
		Scan(ctx, &m)
	require.NoError(t, err)
	switch v := m["num"]; v.(type) {
	case int32:
		require.Equal(t, int32(10), v)
	case int64:
		require.Equal(t, int64(10), v)
	default:
		t.Fail()
	}
}

func testSelectMapSlice(t *testing.T, db *bun.DB) {
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

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

	err = db.NewSelect().TableExpr("(SELECT 42 AS num) AS t").Where("1 = 2").Scan(ctx, model)
	require.Equal(t, sql.ErrNoRows, err)

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
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

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
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

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
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

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

func testSelectJSONMap(t *testing.T, db *bun.DB) {
	type Model struct {
		Map map[string]string
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("? AS map", map[string]string{"hello": "world"}).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"hello": "world"}, model.Map)

	err = db.NewSelect().
		ColumnExpr("NULL AS map").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), model.Map)
}

func testSelectJSONStruct(t *testing.T, db *bun.DB) {
	type Struct struct {
		Hello string
	}

	type Model struct {
		Struct Struct
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("? AS struct", Struct{Hello: "world"}).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, Struct{Hello: "world"}, model.Struct)

	err = db.NewSelect().
		ColumnExpr("NULL AS struct").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, Struct{}, model.Struct)
}

func testSelectRawMessage(t *testing.T, db *bun.DB) {
	type Model struct {
		Raw json.RawMessage
	}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("? AS raw", map[string]string{"hello": "world"}).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, `{"hello":"world"}`, string(model.Raw))

	err = db.NewSelect().
		ColumnExpr("NULL AS raw").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Nil(t, model.Raw)
}

func testScanNullVar(t *testing.T, db *bun.DB) {
	num := int(42)
	err := db.NewSelect().ColumnExpr("NULL").Scan(ctx, &num)
	require.NoError(t, err)
	require.Zero(t, num)
}

func testScanSingleRow(t *testing.T, db *bun.DB) {
	rows, err := db.QueryContext(ctx, "SELECT 42")
	require.NoError(t, err)
	defer rows.Close()

	if !rows.Next() {
		t.Fail()
	}

	var num int
	err = db.ScanRow(ctx, rows, &num)
	require.NoError(t, err)
	require.Equal(t, 42, num)
}

func testScanSingleRowByRow(t *testing.T, db *bun.DB) {
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

	values := db.NewValues(&[]map[string]interface{}{
		{"num": 1},
		{"num": 2},
		{"num": 3},
	})

	rows, err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		OrderExpr("t.num DESC").
		Rows(ctx)
	require.NoError(t, err)
	defer rows.Close()

	var nums []int

	for rows.Next() {
		var num int

		err := db.ScanRow(ctx, rows, &num)
		require.NoError(t, err)

		nums = append(nums, num)
	}

	require.NoError(t, rows.Err())
	require.Equal(t, []int{3, 2, 1}, nums)
}

func testScanRows(t *testing.T, db *bun.DB) {
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
	}

	values := db.NewValues(&[]map[string]interface{}{
		{"num": 1},
		{"num": 2},
		{"num": 3},
	})

	rows, err := db.NewSelect().
		With("t", values).
		TableExpr("t").
		OrderExpr("t.num DESC").
		Rows(ctx)
	require.NoError(t, err)
	defer rows.Close()

	var nums []int
	err = db.ScanRows(ctx, rows, &nums)
	require.NoError(t, err)
	require.Equal(t, []int{3, 2, 1}, nums)
}

func testRunInTx(t *testing.T, db *bun.DB) {
	type Counter struct {
		Count int64
	}

	mustResetModel(t, ctx, db, (*Counter)(nil))

	_, err := db.NewInsert().Model(&Counter{Count: 0}).Exec(ctx)
	require.NoError(t, err)

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*Counter)(nil)).
			Set("count = count + 1").
			Where("1 = 1").
			Exec(ctx)
		return err
	})
	require.NoError(t, err)

	var count int
	err = db.NewSelect().Model((*Counter)(nil)).Scan(ctx, &count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model((*Counter)(nil)).
			Set("count = count + 1").
			Where("TRUE").
			Exec(ctx); err != nil {
			return err
		}
		return errors.New("rollback")
	})
	require.Error(t, err)

	err = db.NewSelect().Model((*Counter)(nil)).Scan(ctx, &count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func testJSONSpecialChars(t *testing.T, db *bun.DB) {
	type Model struct {
		ID    int                    `bun:",pk,autoincrement"`
		Attrs map[string]interface{} `bun:"type:json"`
	}

	ctx := context.Background()

	mustResetModel(t, ctx, db, (*Model)(nil))

	model := &Model{
		Attrs: map[string]interface{}{
			"hello": "\000world\nworld\u0000",
		},
	}
	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)

	model = new(Model)
	err = db.NewSelect().Model(model).Scan(ctx)
	require.NoError(t, err)
	switch db.Dialect().Name() {
	case dialect.MySQL:
		require.Equal(t, map[string]interface{}{
			"hello": "\x00world\nworld\x00",
		}, model.Attrs)
	default:
		require.Equal(t, map[string]interface{}{
			"hello": "\\u0000world\nworld\\u0000",
		}, model.Attrs)
	}
}

func testJSONInterface(t *testing.T, db *bun.DB) {
	type Model struct {
		ID    int         `bun:",pk,autoincrement"`
		Value interface{} `bun:"type:json"`
	}

	ctx := context.Background()

	mustResetModel(t, ctx, db, (*Model)(nil))

	model := new(Model)
	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)

	model = &Model{
		Value: "hello",
	}
	_, err = db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)
}

type JSONValue struct {
	str string
}

var _ driver.Valuer = (*JSONValue)(nil)

func (v *JSONValue) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		v.str = string(src)
	case string:
		v.str = src
	default:
		panic("not reached")
	}
	return nil
}

func (v *JSONValue) Value() (driver.Value, error) {
	return `"driver.Value"`, nil
}

func testJSONValuer(t *testing.T, db *bun.DB) {
	type Model struct {
		ID    int       `bun:",pk,autoincrement"`
		Value JSONValue `bun:"type:json"`
	}

	ctx := context.Background()

	mustResetModel(t, ctx, db, (*Model)(nil))

	model := new(Model)
	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, `"driver.Value"`, model2.Value.str)
}

func testSelectBool(t *testing.T, db *bun.DB) {
	var flag bool
	err := db.NewSelect().ColumnExpr("1").Scan(ctx, &flag)
	require.NoError(t, err)
	require.True(t, flag)

	err = db.NewSelect().ColumnExpr("0").Scan(ctx, &flag)
	require.NoError(t, err)
	require.False(t, flag)
}

func testRawQuery(t *testing.T, db *bun.DB) {
	var num int
	err := db.NewRaw("SELECT ?", 123).Scan(ctx, &num)

	require.NoError(t, err)
	require.Equal(t, 123, num)

	_ = db.RunInTx(context.Background(), nil, func(ctx context.Context, tx bun.Tx) error {
		var num int
		err := db.NewRaw("SELECT ?", 456).Scan(ctx, &num)
		require.NoError(t, err)
		require.Equal(t, 456, num)
		return nil
	})

	conn, err := db.Conn(context.Background())
	require.NoError(t, err)

	err = conn.NewRaw("SELECT ?", 789).Scan(ctx, &num)
	require.NoError(t, err)
	require.Equal(t, 789, num)
}

func testFKViolation(t *testing.T, db *bun.DB) {
	type Deck struct {
		ID     int `bun:",pk,autoincrement"`
		UserID int
	}

	type User struct {
		ID int `bun:",pk,autoincrement"`
	}

	if db.Dialect().Name() == dialect.SQLite {
		_, err := db.Exec("PRAGMA foreign_keys = ON;")
		require.NoError(t, err)
	}

	for _, model := range []interface{}{(*Deck)(nil), (*User)(nil)} {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)
	}

	mustResetModel(t, ctx, db, (*User)(nil))
	_, err := db.NewCreateTable().
		Model((*Deck)(nil)).
		IfNotExists().
		ForeignKey("(user_id) REFERENCES users (id) ON DELETE CASCADE").
		Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Deck)(nil))

	// Empty deck should violate FK constraint.
	_, err = db.NewInsert().Model(new(Deck)).Exec(ctx)
	require.Error(t, err)

	// Create a deck that violates the user_id FK constraint
	deck := &Deck{UserID: 42}

	_, err = db.NewInsert().Model(deck).Exec(ctx)
	require.Error(t, err)

	decks := []*Deck{deck}
	_, err = db.NewInsert().Model(&decks).Exec(ctx)
	require.Error(t, err)

	n, err := db.NewSelect().Model((*Deck)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func testWithForeignKeysAndRules(t *testing.T, db *bun.DB) {
	type User struct {
		ID   int    `bun:",pk"`
		Type string `bun:",pk"`
		Name string
	}
	type Deck struct {
		ID       int `bun:",pk"`
		UserID   int
		UserType string
		User     *User `bun:"rel:belongs-to,join:user_id=id,join:user_type=type,on_update:cascade,on_delete:set null"`
	}

	if db.Dialect().Name() == dialect.SQLite {
		_, err := db.Exec("PRAGMA foreign_keys = ON;")
		require.NoError(t, err)
	}

	for _, model := range []interface{}{(*Deck)(nil), (*User)(nil)} {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)
	}

	mustResetModel(t, ctx, db, (*User)(nil))
	_, err := db.NewCreateTable().
		Model((*Deck)(nil)).
		IfNotExists().
		WithForeignKeys().
		Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Deck)(nil))

	// Empty deck should violate FK constraint.
	_, err = db.NewInsert().Model(new(Deck)).Exec(ctx)
	require.Error(t, err)

	// Create a deck that violates the user_id FK constraint
	deck := &Deck{UserID: 42}

	_, err = db.NewInsert().Model(deck).Exec(ctx)
	require.Error(t, err)

	decks := []*Deck{deck}
	_, err = db.NewInsert().Model(&decks).Exec(ctx)
	require.Error(t, err)

	n, err := db.NewSelect().Model((*Deck)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	_, err = db.NewInsert().Model(&User{ID: 1, Type: "admin", Name: "root"}).Exec(ctx)
	require.NoError(t, err)
	res, err := db.NewInsert().Model(&Deck{UserID: 1, UserType: "admin"}).Exec(ctx)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	// Update User ID and check for FK update
	res, err = db.NewUpdate().Model(&User{}).Where("id = ?", 1).Where("type = ?", "admin").Set("id = ?", 2).Exec(ctx)
	require.NoError(t, err)

	affected, err = res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	n, err = db.NewSelect().Model(&Deck{}).Where("user_id = 1").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = db.NewSelect().Model(&Deck{}).Where("user_id = 2").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	// Delete user and check for FK delete
	_, err = db.NewDelete().Model(&User{}).Where("id = ?", 2).Exec(ctx)
	require.NoError(t, err)

	n, err = db.NewSelect().Model(&Deck{}).Where("user_id = 2").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func testWithForeignKeys(t *testing.T, db *bun.DB) {
	type User struct {
		ID   int    `bun:",pk,autoincrement"`
		Type string `bun:",pk"`
		Name string
	}
	type Deck struct {
		ID       int `bun:",pk,autoincrement"`
		UserID   int
		UserType string
		User     *User `bun:"rel:belongs-to,join:user_id=id,join:user_type=type"`
	}

	if db.Dialect().Name() == dialect.SQLite {
		_, err := db.Exec("PRAGMA foreign_keys = ON;")
		require.NoError(t, err)
	}

	for _, model := range []interface{}{(*Deck)(nil), (*User)(nil)} {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)
	}

	mustResetModel(t, ctx, db, (*User)(nil))

	_, err := db.NewCreateTable().
		Model((*Deck)(nil)).
		IfNotExists().
		WithForeignKeys().
		Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Deck)(nil))

	// Empty deck should violate FK constraint.
	_, err = db.NewInsert().Model(new(Deck)).Exec(ctx)
	require.Error(t, err)

	// Create a deck that violates the user_id FK constraint
	deck := &Deck{UserID: 42}

	_, err = db.NewInsert().Model(deck).Exec(ctx)
	require.Error(t, err)

	decks := []*Deck{deck}
	_, err = db.NewInsert().Model(&decks).Exec(ctx)
	require.Error(t, err)

	n, err := db.NewSelect().Model((*Deck)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	_, err = db.NewInsert().Model(&User{ID: 1, Type: "admin", Name: "root"}).Exec(ctx)
	require.NoError(t, err)
	res, err := db.NewInsert().Model(&Deck{UserID: 1, UserType: "admin"}).Exec(ctx)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	// Select with Relation should work
	d := Deck{}
	err = db.NewSelect().Model(&d).Relation("User").Scan(ctx)
	require.NoError(t, err)
	require.NotNil(t, d.User)
	require.Equal(t, d.User.Name, "root")
}

func testWithForeignKeysHasMany(t *testing.T, db *bun.DB) {
	type User struct {
		ID     int `bun:",pk"`
		DeckID int
		Name   string
	}
	type Deck struct {
		ID    int     `bun:",pk"`
		Users []*User `bun:"rel:has-many,join:id=deck_id"`
	}

	if db.Dialect().Name() == dialect.SQLite {
		_, err := db.Exec("PRAGMA foreign_keys = ON;")
		require.NoError(t, err)
	}

	for _, model := range []interface{}{(*Deck)(nil), (*User)(nil)} {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)
	}

	mustResetModel(t, ctx, db, (*User)(nil))
	_, err := db.NewCreateTable().
		Model((*Deck)(nil)).
		IfNotExists().
		WithForeignKeys().
		Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Deck)(nil))

	deckID := 1
	deck := Deck{ID: deckID}
	_, err = db.NewInsert().Model(&deck).Exec(ctx)
	require.NoError(t, err)

	userID1 := 1
	userID2 := 2
	users := []*User{
		{ID: userID1, DeckID: deckID, Name: "user 1"},
		{ID: userID2, DeckID: deckID, Name: "user 2"},
	}

	res, err := db.NewInsert().Model(&users).Exec(ctx)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)

	err = db.NewSelect().Model(&deck).Relation("Users").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, deck.Users, 2)
}

func testWithPointerForeignKeysHasMany(t *testing.T, db *bun.DB) {
	type User struct {
		ID     *int `bun:",pk"`
		DeckID *int
		Name   string
	}
	type Deck struct {
		ID    *int    `bun:",pk"`
		Users []*User `bun:"rel:has-many,join:id=deck_id"`
	}

	if db.Dialect().Name() == dialect.SQLite {
		_, err := db.Exec("PRAGMA foreign_keys = ON;")
		require.NoError(t, err)
	}

	for _, model := range []interface{}{(*Deck)(nil), (*User)(nil)} {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)
	}

	mustResetModel(t, ctx, db, (*User)(nil))
	_, err := db.NewCreateTable().
		Model((*Deck)(nil)).
		IfNotExists().
		WithForeignKeys().
		Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Deck)(nil))

	deckID := 1
	deck := Deck{ID: &deckID}
	_, err = db.NewInsert().Model(&deck).Exec(ctx)
	require.NoError(t, err)

	userID1 := 1
	userID2 := 2
	users := []*User{
		{ID: &userID1, DeckID: &deckID, Name: "user 1"},
		{ID: &userID2, DeckID: &deckID, Name: "user 2"},
	}

	res, err := db.NewInsert().Model(&users).Exec(ctx)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)

	err = db.NewSelect().Model(&deck).Relation("Users").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, deck.Users, 2)
}

func testWithPointerForeignKeysHasManyWithDriverValuer(t *testing.T, db *bun.DB) {
	type User struct {
		ID     *int `bun:",pk"`
		DeckID sql.NullInt64
		Name   string
	}
	type Deck struct {
		ID    int64   `bun:",pk"`
		Users []*User `bun:"rel:has-many,join:id=deck_id"`
	}

	if db.Dialect().Name() == dialect.SQLite {
		_, err := db.Exec("PRAGMA foreign_keys = ON;")
		require.NoError(t, err)
	}

	for _, model := range []interface{}{(*Deck)(nil), (*User)(nil)} {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)
	}

	mustResetModel(t, ctx, db, (*User)(nil))
	_, err := db.NewCreateTable().
		Model((*Deck)(nil)).
		IfNotExists().
		WithForeignKeys().
		Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Deck)(nil))

	deckID := int64(1)
	deck := Deck{ID: deckID}
	_, err = db.NewInsert().Model(&deck).Exec(ctx)
	require.NoError(t, err)

	userID1 := 1
	userID2 := 2
	users := []*User{
		{ID: &userID1, DeckID: sql.NullInt64{Int64: deckID, Valid: true}, Name: "user 1"},
		{ID: &userID2, DeckID: sql.NullInt64{Int64: deckID, Valid: true}, Name: "user 2"},
	}

	res, err := db.NewInsert().Model(&users).Exec(ctx)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)

	err = db.NewSelect().Model(&deck).Relation("Users").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, deck.Users, 2)
}

func testInterfaceAny(t *testing.T, db *bun.DB) {
	switch db.Dialect().Name() {
	case dialect.MySQL:
		t.Skip()
	}

	type Model struct {
		Value interface{}
	}

	model := new(Model)
	err := db.NewSelect().ColumnExpr("NULL AS value").Scan(ctx, model)
	require.NoError(t, err)
	require.Nil(t, model.Value)

	model = new(Model)
	err = db.NewSelect().ColumnExpr(`'hello' AS value`).Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, "hello", model.Value)

	model = new(Model)
	err = db.NewSelect().ColumnExpr(`42 AS value`).Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, int64(42), model.Value)
}

func testInterfaceJSON(t *testing.T, db *bun.DB) {
	type Model struct {
		Value interface{} `bun:"type:json"`
	}

	model := new(Model)
	err := db.NewSelect().ColumnExpr("NULL AS value").Scan(ctx, model)
	require.NoError(t, err)
	require.Nil(t, model.Value)

	model = new(Model)
	err = db.NewSelect().ColumnExpr(`'"hello"' AS value`).Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, "hello", model.Value)
}

func testScanRawMessage(t *testing.T, db *bun.DB) {
	type Model struct {
		ID    int64 `bun:",pk,autoincrement"`
		Value json.RawMessage
	}

	ctx := context.Background()

	mustResetModel(t, ctx, db, (*Model)(nil))

	models := []Model{
		{Value: json.RawMessage(`"hello"`)},
		{Value: json.RawMessage(`"world"`)},
	}
	_, err := db.NewInsert().Model(&models).Exec(ctx)
	require.NoError(t, err)

	var models1 []Model
	err = db.NewSelect().Model(&models1).Order("id ASC").Scan(ctx)
	require.NoError(t, err)

	var models2 []Model
	err = db.NewSelect().Model(&models2).Order("id DESC").Scan(ctx)
	require.NoError(t, err)

	require.Equal(t, models, models1)
}

func testPointers(t *testing.T, db *bun.DB) {
	type Model struct {
		ID  *int64 `bun:",default:0"`
		Str *string
	}

	ctx := context.Background()

	for _, id := range []int64{-1, 0, 1} {
		mustResetModel(t, ctx, db, (*Model)(nil))

		var model Model
		if id >= 0 {
			str := "hello"
			model.ID = &id
			model.Str = &str

		}

		_, err := db.NewInsert().Model(&model).Exec(ctx)
		require.NoError(t, err)

		var model2 Model
		err = db.NewSelect().Model(&model2).Order("id ASC").Scan(ctx)
		require.NoError(t, err)
	}
}

func testExists(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	exists, err := db.NewSelect().ColumnExpr("1").Exists(ctx)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = db.NewSelect().ColumnExpr("1").Where("1 = 2").Exists(ctx)
	require.NoError(t, err)
	require.False(t, exists)
}

func testScanTimeIntoString(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	var str string
	err := db.NewSelect().ColumnExpr("CURRENT_TIMESTAMP").Scan(ctx, &str)
	require.NoError(t, err)
	require.NotZero(t, str)
}

func testModelNonPointer(t *testing.T, db *bun.DB) {
	type Model struct{}

	_, err := db.NewInsert().Model(Model{}).ExcludeColumn("id").Returning("id").Exec(ctx)
	require.Error(t, err)
	require.Equal(t, "bun: Model(non-pointer dbtest_test.Model)", err.Error())
}

func testBinaryData(t *testing.T, db *bun.DB) {
	type Model struct {
		ID   int64 `bun:",pk,autoincrement"`
		Data []byte
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	_, err := db.NewInsert().Model(&Model{Data: []byte("hello")}).Exec(ctx)
	require.NoError(t, err)

	var model Model
	err = db.NewSelect().Model(&model).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), model.Data)
}

func testUpsert(t *testing.T, db *bun.DB) {
	if db.Dialect().Name() == dialect.MSSQL {
		t.Skip("mssql")
	}

	type Model struct {
		ID  int64 `bun:",pk,autoincrement"`
		Str string
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	model := &Model{ID: 1, Str: "hello"}

	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)

	model.Str = "world"

	switch db.Dialect().Name() {
	case dialect.MySQL:
		_, err := db.NewInsert().Model(model).On("DUPLICATE KEY UPDATE").Exec(ctx)
		require.NoError(t, err)
	default:
		_, err := db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
		require.NoError(t, err)
	}

	err = db.NewSelect().Model(model).WherePK().Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, "world", model.Str)
}

func testMultiUpdate(t *testing.T, db *bun.DB) {
	if !db.Dialect().Features().Has(feature.CTE) {
		t.Skip()
		return
	}

	type Model struct {
		ID  int64 `bun:",pk,autoincrement"`
		Str string
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	model := &Model{ID: 1, Str: "hello"}

	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)

	selq := db.NewSelect().Model(new(Model))

	_, err = db.NewUpdate().
		With("src", selq).
		TableExpr("models").
		Table("src").
		Set("? = src.str", db.UpdateFQN("models", "str")).
		Where("models.id = src.id").
		Exec(ctx)
	require.NoError(t, err)
}

func testUpdateWithSkipupdateTag(t *testing.T, db *bun.DB) {
	type Model struct {
		ID        int64 `bun:",pk,autoincrement"`
		Name      string
		CreatedAt time.Time `bun:",skipupdate"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	createdAt := time.Now().Truncate(time.Minute).UTC()

	model := &Model{ID: 1, Name: "foo", CreatedAt: createdAt}

	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)
	require.NotZero(t, model.CreatedAt)

	//
	// update field with tag "skipupdate"
	//
	model.CreatedAt = model.CreatedAt.Add(2 * time.Minute)
	_, err = db.NewUpdate().Model(model).WherePK().Exec(ctx)
	require.NoError(t, err)

	//
	// check
	//
	model_ := new(Model)
	model_.ID = model.ID
	err = db.NewSelect().Model(model_).WherePK().Scan(ctx)
	require.NoError(t, err, "select")
	require.NotEmpty(t, model_)
	require.Equal(t, model.ID, model_.ID)
	require.Equal(t, model.Name, model_.Name)
	require.Equal(t, createdAt.UTC(), model_.CreatedAt.UTC())

	require.NotEqual(t, model.CreatedAt.UTC(), model_.CreatedAt.UTC())
}

func testScanAndCount(t *testing.T, db *bun.DB) {
	type Model struct {
		ID  int64 `bun:",pk,autoincrement"`
		Str string
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	t.Run("tx", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				var models []Model
				count, err := tx.NewSelect().Model(&models).ScanAndCount(ctx)
				require.NoError(t, err)
				require.Equal(t, 0, count)
				return err
			})
			require.NoError(t, err)
		}
	})

	t.Run("no limit", func(t *testing.T) {
		src := []Model{
			{Str: "str1"},
			{Str: "str2"},
		}
		_, err := db.NewInsert().Model(&src).Exec(ctx)
		require.NoError(t, err)

		var dest []Model
		count, err := db.NewSelect().Model(&dest).ScanAndCount(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 2, len(dest))
	})
}

func testEmbedModelValue(t *testing.T, db *bun.DB) {
	type DoubleEmbed struct {
		A string
		B string
	}
	type Embed struct {
		Foo string
		Bar string
		C   DoubleEmbed `bun:"embed:c_"`
		D   DoubleEmbed `bun:"embed:d_"`
	}
	type Model struct {
		X Embed `bun:"embed:x_"`
		Y Embed `bun:"embed:y_"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	m1 := &Model{
		X: Embed{
			Foo: "x.foo",
			Bar: "x.bar",
			C: DoubleEmbed{
				A: "x.c.a",
				B: "x.c.b",
			},
			D: DoubleEmbed{
				A: "x.d.a",
				B: "x.d.b",
			},
		},
		Y: Embed{
			Foo: "y.foo",
			Bar: "y.bar",
			C: DoubleEmbed{
				A: "y.c.a",
				B: "y.c.b",
			},
			D: DoubleEmbed{
				A: "y.d.a",
				B: "y.d.b",
			},
		},
	}
	_, err := db.NewInsert().Model(m1).Exec(ctx)
	require.NoError(t, err)

	var m2 Model
	err = db.NewSelect().Model(&m2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, *m1, m2)
}

func testEmbedModelPointer(t *testing.T, db *bun.DB) {
	type DoubleEmbed struct {
		A string
		B string
	}
	type Embed struct {
		Foo string
		Bar string
		C   *DoubleEmbed `bun:"embed:c_"`
		D   *DoubleEmbed `bun:"embed:d_"`
	}
	type Model struct {
		X *Embed `bun:"embed:x_"`
		Y *Embed `bun:"embed:y_"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	m1 := &Model{
		X: &Embed{
			Foo: "x.foo",
			Bar: "x.bar",
			C: &DoubleEmbed{
				A: "x.c.a",
				B: "x.c.b",
			},
			D: &DoubleEmbed{
				A: "x.d.a",
				B: "x.d.b",
			},
		},
		Y: &Embed{
			Foo: "y.foo",
			Bar: "y.bar",
			C: &DoubleEmbed{
				A: "y.c.a",
				B: "y.c.b",
			},
			D: &DoubleEmbed{
				A: "y.d.a",
				B: "y.d.b",
			},
		},
	}
	_, err := db.NewInsert().Model(m1).Exec(ctx)
	require.NoError(t, err)

	var m2 Model
	err = db.NewSelect().Model(&m2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, *m1, m2)
}

func testEmbedTypeField(t *testing.T, db *bun.DB) {
	type Embed string
	type Model struct {
		Embed
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	m1 := &Model{
		Embed: Embed("foo"),
	}
	_, err := db.NewInsert().Model(m1).Exec(ctx)
	require.NoError(t, err)

	var m2 Model
	err = db.NewSelect().Model(&m2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, *m1, m2)
}

type JSONField struct {
	Foo string `json:"foo"`
}

func (f *JSONField) MarshalJSON() ([]byte, error) {
	return []byte(`{"foo": "bar"}`), nil
}

func testJSONMarshaler(t *testing.T, db *bun.DB) {
	type Model struct {
		Field *JSONField
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	m1 := &Model{Field: new(JSONField)}
	_, err := db.NewInsert().Model(m1).Exec(ctx)
	require.NoError(t, err)

	var m2 Model
	err = db.NewSelect().Model(&m2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, "bar", m2.Field.Foo)
}

type DriverValue struct {
	s string
}

var _ driver.Valuer = (*DriverValue)(nil)

func (v *DriverValue) Value() (driver.Value, error) {
	return v.s, nil
}

func testNilDriverValue(t *testing.T, db *bun.DB) {
	type Model struct {
		Value *DriverValue `bun:"type:varchar(100)"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	_, err := db.NewInsert().Model(&Model{}).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&Model{Value: &DriverValue{s: "hello"}}).Exec(ctx)
	require.NoError(t, err)
}

func testRunInTxAndSavepoint(t *testing.T, db *bun.DB) {
	type Counter struct {
		Count int64
	}

	mustResetModel(t, ctx, db, (*Counter)(nil))

	_, err := db.NewInsert().Model(&Counter{Count: 0}).Exec(ctx)
	require.NoError(t, err)

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.RunInTx(ctx, nil, func(ctx context.Context, sp bun.Tx) error {
			_, err := sp.NewUpdate().Model((*Counter)(nil)).
				Set("count = count + 1").
				Where("1 = 1").
				Exec(ctx)
			return err
		})
		require.NoError(t, err)
		// rolling back the transaction should rollback what happened inside savepoint
		return errors.New("fake error")
	})
	require.Error(t, err)

	var count int
	err = db.NewSelect().Model((*Counter)(nil)).Scan(ctx, &count)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.RunInTx(ctx, nil, func(ctx context.Context, sp bun.Tx) error {
			_, err := sp.NewInsert().Model(&Counter{Count: 1}).
				Exec(ctx)
			require.NoError(t, err)
			return err
		})
		require.NoError(t, err)

		// ignored on purpose this error
		// rolling back a savepoint should not affect the transaction
		// nor other savepoints on the same level
		_ = tx.RunInTx(ctx, nil, func(ctx context.Context, sp bun.Tx) error {
			_, err := sp.NewInsert().Model(&Counter{Count: 2}).
				Exec(ctx)
			require.NoError(t, err)
			return errors.New("fake error")
		})

		return err
	})
	require.NoError(t, err)

	count, err = db.NewSelect().Model((*Counter)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = db.ResetModel(ctx, (*Counter)(nil))
	require.NoError(t, err)

	// happy path, commit transaction, savepoints and sub-savepoints
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(&Counter{Count: 1}).
			Exec(ctx)
		require.NoError(t, err)

		err = tx.RunInTx(ctx, nil, func(ctx context.Context, sp bun.Tx) error {
			_, err := sp.NewInsert().Model(&Counter{Count: 1}).
				Exec(ctx)
			if err != nil {
				return err
			}

			return sp.RunInTx(ctx, nil, func(ctx context.Context, subSp bun.Tx) error {
				_, err := subSp.NewInsert().Model(&Counter{Count: 1}).
					Exec(ctx)
				return err
			})
		})
		require.NoError(t, err)

		err = tx.RunInTx(ctx, nil, func(ctx context.Context, sp bun.Tx) error {
			_, err := sp.NewInsert().Model(&Counter{Count: 2}).
				Exec(ctx)
			return err
		})

		return err
	})
	require.NoError(t, err)

	count, err = db.NewSelect().Model((*Counter)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, count)
}

type anotherString string

var _ driver.Valuer = (*anotherString)(nil)

func (v anotherString) Value() (driver.Value, error) {
	return v, nil
}

func testDriverValuerReturnsItself(t *testing.T, db *bun.DB) {
	expectedValue := anotherString("example value")

	type Model struct {
		ID    int           `bun:",pk,autoincrement"`
		Value anotherString `bun:"value"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	model := &Model{Value: expectedValue}
	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.Error(t, err)
}

func testNoPanicWhenReturningNullColumns(t *testing.T, db *bun.DB) {
	type Model struct {
		Value     string  `bun:"value,notnull"`
		NullValue *string `bun:"null_value"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*Model)(nil))

	modelSlice := []*Model{{Value: "boom"}}

	require.NotPanics(t, func() {
		db.NewInsert().Model(&modelSlice).Exec(ctx)
	})
}

func testNoForeignKeyForPrimaryKey(t *testing.T, db *bun.DB) {
	inspect := inspectDbOrSkip(t, db)

	for _, tt := range []struct {
		name     string
		model    interface{}
		dontWant sqlschema.ForeignKey
	}{
		{name: "has-one relation", model: (*struct {
			bun.BaseModel `bun:"table:users"`
			ID            string `bun:",pk"`

			Profile *struct {
				bun.BaseModel `bun:"table:profiles"`
				ID            string `bun:",pk"`
				UserID        string
			} `bun:"rel:has-one,join:id=user_id"`
		})(nil), dontWant: sqlschema.ForeignKey{
			From: sqlschema.NewColumnReference("users", "id"),
			To:   sqlschema.NewColumnReference("profiles", "user_id"),
		}},

		{name: "belongs-to relation", model: (*struct {
			bun.BaseModel `bun:"table:profiles"`
			ID            string `bun:",pk"`

			User *struct {
				bun.BaseModel `bun:"table:users"`
				ID            string `bun:",pk"`
				ProfileID     string
			} `bun:"rel:belongs-to,join:id=profile_id"`
		})(nil), dontWant: sqlschema.ForeignKey{
			From: sqlschema.NewColumnReference("profiles", "id"),
			To:   sqlschema.NewColumnReference("users", "profile_id"),
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mustDropTableOnCleanup(t, ctx, db, tt.model)

			_, err := db.NewCreateTable().Model(tt.model).WithForeignKeys().Exec(ctx)
			require.NoError(t, err, "create table")

			state := inspect(ctx)
			require.NotContainsf(t, state.ForeignKeys, tt.dontWant,
				"%s.%s -> %s.%s is not inteded",
				tt.dontWant.From.TableName, tt.dontWant.From.Column,
				tt.dontWant.To.TableName, tt.dontWant.To.Column,
			)
		})
	}
}

func mustResetModel(tb testing.TB, ctx context.Context, db *bun.DB, models ...interface{}) {
	err := db.ResetModel(ctx, models...)
	require.NoError(tb, err, "must reset model")
	mustDropTableOnCleanup(tb, ctx, db, models...)
}

func mustDropTableOnCleanup(tb testing.TB, ctx context.Context, db *bun.DB, models ...interface{}) {
	tb.Cleanup(func() {
		for _, model := range models {
			drop := db.NewDropTable().IfExists().Cascade().Model(model)
			_, err := drop.Exec(ctx)
			require.NoError(tb, err, "must drop table: %q", drop.GetTableName())
		}
	})
}

func TestConnResolver(t *testing.T) {
	dsn := os.Getenv("PG")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/test?sslmode=disable"
	}

	rwdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	t.Cleanup(func() {
		require.NoError(t, rwdb.Close())
	})

	rodb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	t.Cleanup(func() {
		require.NoError(t, rodb.Close())
	})

	resolver := bunexp.NewReadWriteConnResolver(
		bunexp.WithDBReplica(rodb, bunexp.DBReplicaReadOnly),
	)

	db := bun.NewDB(rwdb, pgdialect.New(), bun.WithConnResolver(resolver))
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		bundebug.FromEnv(),
	))

	var num int
	err := db.NewSelect().ColumnExpr("1").Scan(ctx, &num)
	require.NoError(t, err)
	require.Equal(t, 1, num)
	require.GreaterOrEqual(t, rodb.Stats().OpenConnections, 1)
	require.Equal(t, 0, rwdb.Stats().OpenConnections)
}

type doNotCompare [0]func()

type notCompareKey struct {
	doNotCompare
	s string
}

func (x *notCompareKey) AppendQuery(_ schema.Formatter, b []byte) ([]byte, error) {
	b = append(b, '\'')
	b = append(b, x.s...)
	return append(b, '\''), nil
}

func (x *notCompareKey) Value() (driver.Value, error) { return x.s, nil }
func (x *notCompareKey) FromUUID(u uuid.UUID)         { x.s = u.String() }

func (x *notCompareKey) Scan(src any) error {
	var u uuid.UUID
	if err := u.Scan(src); err != nil {
		return err
	}
	x.FromUUID(u)
	return nil
}

func testWithPointerPrimaryKeyHasManyWithDriverValuer(t *testing.T, db *bun.DB) {
	type Item struct {
		ID        *notCompareKey `bun:"type:VARCHAR,pk"`
		ProgramID *notCompareKey `bun:"type:VARCHAR"`
		Name      string
	}
	type Program struct {
		ID    *notCompareKey `bun:"type:VARCHAR,pk"`
		Items []*Item        `bun:"rel:has-many,join:id=program_id"`
	}

	mustResetModel(t, ctx, db, (*Item)(nil), (*Program)(nil))

	programID := &notCompareKey{s: uuid.New().String()}
	program := Program{ID: programID}

	_, err := db.NewInsert().Model(&program).Exec(ctx)
	require.NoError(t, err)

	itemID1 := &notCompareKey{s: uuid.New().String()}
	itemID2 := &notCompareKey{s: uuid.New().String()}
	users := []*Item{
		{ID: itemID1, ProgramID: programID, Name: "Item 1"},
		{ID: itemID2, ProgramID: programID, Name: "Item 2"},
	}

	res, err := db.NewInsert().Model(&users).Exec(ctx)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)

	err = db.NewSelect().Model(&program).Relation("Items").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, program.Items, 2)
}
