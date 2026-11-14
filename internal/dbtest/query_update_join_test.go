package dbtest_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type updateJoinGoods struct {
	bun.BaseModel `bun:"table:shop_goods"`
	Id            int64      `bun:"id,pk"`
	ShopId        int64      `bun:"shop_id"`
	DeletedAt     *time.Time `bun:"deleted_at,nullzero,soft_delete"`
}

type updateJoinSku struct {
	bun.BaseModel `bun:"table:shop_goods_sku"`
	GoodsId       int64      `bun:"goods_id,pk"`
	SkuId         int64      `bun:"sku_id,pk"`
	Stock         int64      `bun:"stock"`
	DeletedAt     *time.Time `bun:"deleted_at,nullzero,soft_delete"`
}

func goodsSub(db *bun.DB) *bun.SelectQuery {
	return db.NewSelect().Model(&updateJoinGoods{}).
		Where("id IN (?)", bun.List([]int64{9})).
		Where("shop_id = ?", 1).
		WhereDeleted().
		Column("id", "deleted_at")
}

func appendUpdate(t *testing.T, q *bun.UpdateQuery, db *bun.DB) string {
	t.Helper()
	s, err := q.AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)
	return string(s)
}

// On dialects without multi-table UPDATE (PostgreSQL), an UpdateQuery.Join
// must render as a FROM item with JoinOn/soft-delete conditions moved into
// WHERE, instead of producing the illegal UPDATE ... SET ... JOIN form.
func TestUpdateJoinFirstAsFromItem(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		WhereDeleted().
		Join("(?) AS t1", goodsSub(db)).
		JoinOn("t2.? = t1.? AND t2.? = t1.?",
			bun.Ident("goods_id"), bun.Ident("id"),
			bun.Ident("deleted_at"), bun.Ident("deleted_at")).
		Set("t2.deleted_at = NULL")

	got := appendUpdate(t, q, db)
	require.True(t, strings.Contains(got, ` FROM (SELECT`), got)
	require.True(t, strings.Contains(got, `WHERE (t2."goods_id" = t1."id" AND t2."deleted_at" = t1."deleted_at")`), got)
	require.True(t, strings.HasSuffix(got, `"t2"."deleted_at" IS NOT NULL`), got)
	require.NotContains(t, got, " JOIN")
}

// Inner-join ON conditions may not reference the updated table (a PG
// restriction), so all of them are moved into WHERE.
func TestUpdateJoinSubsequent(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		WhereDeleted().
		Join("(?) AS t1", goodsSub(db)).JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Join("JOIN (?) AS t3", goodsSub(db)).JoinOn("t2.? = t3.?", bun.Ident("goods_id"), bun.Ident("id")).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	// both inner joins render as comma-separated FROM items, join conds in WHERE
	require.Contains(t, got, `) AS t1, (SELECT`, got)
	require.Contains(t, got, `) AS t3 WHERE`, got)
	require.Contains(t, got, `(t2."goods_id" = t1."id") AND (t2."goods_id" = t3."id")`, got)
	require.NotContains(t, got, " ON ", got)
}

// With other tables already present via TableExpr/Table, inner joins still
// render as comma-separated FROM items with their conditions in WHERE.
func TestUpdateJoinWithOtherTables(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		WhereDeleted().
		TableExpr("(?) AS t3", goodsSub(db)).
		Join("(?) AS t1", goodsSub(db)).JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, `) AS t3, (SELECT`, got)
	require.Contains(t, got, `) AS t1 WHERE (t2."goods_id" = t1."id")`, got)
}

// JoinModel as the first join: the joined table's soft-delete condition moves
// into WHERE together with the join conditions.
func TestUpdateJoinModelFirst(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		WhereDeleted().
		JoinModel(&updateJoinGoods{}, "t1").
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, `FROM "shop_goods" AS "t1"`, got)
	require.Contains(t, got, `WHERE (t2."goods_id" = t1."id") AND ("t1"."deleted_at" IS NOT NULL)`, got)
}

// A JoinOnOr join group enters WHERE as one parenthesized group so the OR
// cannot leak into outer conditions.
func TestUpdateJoinOnOr(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		WhereDeleted().
		Join("(?) AS t1", goodsSub(db)).
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		JoinOnOr("t2.? = t1.?", bun.Ident("sku_id"), bun.Ident("id")).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, `WHERE ((t2."goods_id" = t1."id") OR (t2."sku_id" = t1."id"))`, got)
}

// Multi-table UPDATE dialects such as MySQL: joins render after the UPDATE
// table list and before SET.
func TestUpdateJoinMultiTableDialect(t *testing.T) {
	sqldb, err := sql.Open("mysql", "")
	require.NoError(t, err)
	defer sqldb.Close()
	db := bun.NewDB(sqldb, mysqldialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		Join("(?) AS t1", goodsSub(db)).
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Where("t2.? = ?", bun.Ident("sku_id"), 1).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, "AS `t2` JOIN (SELECT", got)
	require.Contains(t, got, "AS t1 ON (t2.`goods_id` = t1.`id`) SET", got)
}

// JoinModel + JoinOnOr: the joined table's soft-delete condition must be
// AND-ed outside the OR group, otherwise in "(c1) OR (c2) AND (sd)" it would
// only constrain the last OR branch.
func TestUpdateJoinOnOrModelSoftDelete(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		WhereDeleted().
		JoinModel(&updateJoinGoods{}, "t1").
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		JoinOnOr("t2.? = t1.?", bun.Ident("stock"), bun.Ident("id")).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got,
		`WHERE ((t2."goods_id" = t1."id") OR (t2."stock" = t1."id")) AND ("t1"."deleted_at" IS NOT NULL)`)
}

// The join-type keyword in a Join argument is matched word-by-word,
// whitespace-separated and case-insensitive (regression).
func TestUpdateJoinKindParsing(t *testing.T) {
	sqldb, err := sql.Open("mysql", "")
	require.NoError(t, err)
	defer sqldb.Close()
	db := bun.NewDB(sqldb, mysqldialect.New())

	q := db.NewUpdate().Model(&updateJoinSku{}).TableAlias("t2").
		Join("  left\njoin\t(?) AS t1", goodsSub(db)).
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Where("t2.? = ?", bun.Ident("sku_id"), 1).
		Set("t2.stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, "AS `t2` LEFT JOIN (SELECT")
	require.Contains(t, got, "AS t1 ON (t2.`goods_id` = t1.`id`) SET")
}

// SelectQuery.Clone must preserve the join kind, otherwise JoinModel renders
// illegal SQL missing the JOIN keyword (regression).
func TestSelectCloneJoinModelKind(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewSelect().Model(&updateJoinSku{}).JoinModel(&updateJoinGoods{}, "t1")
	got, err := q.Clone().AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)
	require.Contains(t, string(got), `JOIN "shop_goods" AS "t1" ON ("t1"."deleted_at" IS NULL)`)
}

// An outer join after an inner join keeps its "<KIND> expr ON cond" form,
// attached to the preceding comma-separated FROM items.
func TestUpdateJoinOuterAfterInner(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
		Join("(?) AS t1", goodsSub(db)).JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Join("LEFT JOIN (?) AS t3", goodsSub(db)).JoinOn("t3.? = t1.?", bun.Ident("id"), bun.Ident("id")).
		WhereDeleted().
		Set("stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, `) AS t1 LEFT JOIN (SELECT`, got)
	require.Contains(t, got, `) AS t3 ON (t3."id" = t1."id") WHERE (t2."goods_id" = t1."id")`, got)
}

// A non-inner join needs a preceding FROM item as its join base; a lone
// outer join must error out instead of producing illegal SQL.
func TestUpdateJoinOuterRequiresFrom(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
		Join("LEFT JOIN (?) AS t1", goodsSub(db)).JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		WhereDeleted().
		Set("stock = 0")

	_, err := q.AppendQuery(db.QueryGen(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires a preceding FROM item")
}

// JoinModel without an explicit alias falls back to the joined model's
// table alias.
func TestUpdateJoinModelDefaultAlias(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	q := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
		JoinModel((*updateJoinGoods)(nil)).
		JoinOn("t2.? = update_join_goods.?", bun.Ident("goods_id"), bun.Ident("id")).
		Set("stock = 0")

	got := appendUpdate(t, q, db)
	require.Contains(t, got, `FROM "shop_goods" AS "update_join_goods"`, got)
	require.Contains(t, got,
		`WHERE (t2."goods_id" = update_join_goods."id") AND ("update_join_goods"."deleted_at" IS NULL)`, got)
}

// An ON clause embedded in the Join string is always rejected, with or
// without args, on every dialect: ON conditions must be added via
// JoinOn/JoinOnOr.
func TestUpdateJoinEmbeddedOnRejected(t *testing.T) {
	sqldb, err := sql.Open("mysql", "")
	require.NoError(t, err)
	defer sqldb.Close()

	for _, db := range []*bun.DB{
		bun.NewDB(nil, pgdialect.New()),
		bun.NewDB(sqldb, mysqldialect.New()),
	} {
		for _, join := range []string{
			"JOIN t1 ON t2.goods_id = t1.id",
			"JOIN t1 oN t2.goods_id = t1.id",
			"LEFT JOIN (?) AS t1 ON t1.id = t2.goods_id",
			"NATURAL JOIN t1 ON t1.id = t2.goods_id",
			"ON t2.goods_id = t1.id",
			"t1 ON",
			"JOIN (?) AS t1 ON t1.id = ?0",
		} {
			q := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
				Join(join, goodsSub(db), 1).
				WhereDeleted().
				Set("stock = 0")
			_, err := q.AppendQuery(db.QueryGen(), nil)
			require.Error(t, err, join)
			require.Contains(t, err.Error(), "ON clause is not allowed", join)
		}
	}
}

// "on" inside string literals, quoted identifiers or comments is not an ON
// clause and must not trigger the embedded-ON rejection.
func TestUpdateJoinOnKeywordFalsePositives(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	for _, join := range []string{
		`/* switch on */ t1`,
		`t1 -- turn on` + "\n",
		`(select 'turn on' as x) AS t1`,
		`"weird on name" AS t1`,
		"monitor",
	} {
		q := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
			Join(join).
			JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
			WhereDeleted().
			Set("stock = 0")
		_, err := q.AppendQuery(db.QueryGen(), nil)
		require.NoError(t, err, join)
	}
}

// SelectQuery.Join / JoinModel output remains unchanged (regression).
func TestSelectJoinUnchanged(t *testing.T) {
	db := bun.NewDB(nil, pgdialect.New())

	s := db.NewSelect().Model(&updateJoinSku{}).
		Join("JOIN (?) AS t1 ON t1.? = shop_goods_sku.?", goodsSub(db), bun.Ident("id"), bun.Ident("goods_id"))
	got, err := s.AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)
	require.Contains(t, string(got), ") AS t1 ON t1."+`"id" = shop_goods_sku."goods_id"`)

	s2 := db.NewSelect().Model(&updateJoinSku{}).JoinModel(&updateJoinGoods{}, "t1")
	got2, err := s2.AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)
	require.Contains(t, string(got2), `JOIN "shop_goods" AS "t1" ON ("t1"."deleted_at" IS NULL)`)
}

//------------------------------------------------------------------------------

// The testXxx cases below are registered in the TestDB batch list in
// db_test.go and execute UPDATE + JOIN against real databases: MySQL uses
// multi-table UPDATE (UPDATE t JOIN u ON ... SET), other dialects use
// UPDATE ... FROM ... WHERE (inner-join conditions moved into WHERE).

// Test data:
//
//	goods: 9(shop1,deleted) 10(shop1,active) 11(shop2,deleted)
//	sku:   1(goods9,deleted) 2(goods10,active) 3(goods11,deleted) 4(goods9,active)
func seedUpdateJoin(t *testing.T, db *bun.DB) {
	mustResetModel(t, ctx, db, (*updateJoinGoods)(nil), (*updateJoinSku)(nil))

	deleted := time.Now()
	goods := []updateJoinGoods{
		{Id: 9, ShopId: 1, DeletedAt: &deleted},
		{Id: 10, ShopId: 1},
		{Id: 11, ShopId: 2, DeletedAt: &deleted},
	}
	skus := []updateJoinSku{
		{GoodsId: 9, SkuId: 1, Stock: 5, DeletedAt: &deleted},
		{GoodsId: 10, SkuId: 2, Stock: 6},
		{GoodsId: 11, SkuId: 3, Stock: 7, DeletedAt: &deleted},
		{GoodsId: 9, SkuId: 4, Stock: 5},
	}
	_, err := db.NewInsert().Model(&goods).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&skus).Exec(ctx)
	require.NoError(t, err)
}

func skuStock(t *testing.T, db *bun.DB, skuId int64) int64 {
	var stock int64
	err := db.NewSelect().Model((*updateJoinSku)(nil)).
		Column("stock").
		Where("sku_id = ?", skuId).
		WhereAllWithDeleted().
		Scan(ctx, &stock)
	require.NoError(t, err)
	return stock
}

func rowsAffected(t *testing.T, res sql.Result) int64 {
	n, err := res.RowsAffected()
	require.NoError(t, err)
	return n
}

// Inner join against a subquery: zero the stock of deleted SKUs whose goods
// are deleted (shop 1).
func testUpdateJoinSubquery(t *testing.T, db *bun.DB) {
	// T-SQL does not allow an AS alias on the UPDATE target.
	if db.Dialect().Name() == dialect.MSSQL {
		t.Skip()
	}
	seedUpdateJoin(t, db)

	res, err := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
		WhereDeleted().
		Join("(?) AS t1", goodsSub(db)).
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Set("stock = 0").
		Exec(ctx)
	require.NoError(t, err)

	require.EqualValues(t, 1, rowsAffected(t, res))
	require.EqualValues(t, 0, skuStock(t, db, 1))
	require.EqualValues(t, 7, skuStock(t, db, 3)) // goods 11 belongs to shop 2, unaffected
}

// JoinModel inner join: only updates visible SKUs whose goods are not deleted
// (the joined table's soft-delete condition applies).
func testUpdateJoinModel(t *testing.T, db *bun.DB) {
	// T-SQL does not allow an AS alias on the UPDATE target.
	if db.Dialect().Name() == dialect.MSSQL {
		t.Skip()
	}
	seedUpdateJoin(t, db)

	res, err := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
		Where("t2.? IS NULL", bun.Ident("deleted_at")).
		JoinModel(&updateJoinGoods{}, "t1").
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		Set("stock = 0").
		Exec(ctx)
	require.NoError(t, err)

	require.EqualValues(t, 1, rowsAffected(t, res))
	require.EqualValues(t, 0, skuStock(t, db, 2))
	require.EqualValues(t, 5, skuStock(t, db, 4)) // goods 9 is deleted, must not match
}

// JoinOnOr + joined-table soft delete: the sd condition must be AND-ed
// outside the OR group, otherwise sku4 would be wrongly updated through the
// first OR branch that matches deleted goods 9.
func testUpdateJoinOnOrSoftDelete(t *testing.T, db *bun.DB) {
	// T-SQL does not allow an AS alias on the UPDATE target.
	if db.Dialect().Name() == dialect.MSSQL {
		t.Skip()
	}
	seedUpdateJoin(t, db)

	res, err := db.NewUpdate().Model((*updateJoinSku)(nil)).TableAlias("t2").
		Where("t2.? IS NULL", bun.Ident("deleted_at")).
		JoinModel(&updateJoinGoods{}, "t1").
		JoinOn("t2.? = t1.?", bun.Ident("goods_id"), bun.Ident("id")).
		JoinOnOr("t2.? = t1.?", bun.Ident("stock"), bun.Ident("shop_id")).
		Set("stock = 99").
		Exec(ctx)
	require.NoError(t, err)

	require.EqualValues(t, 1, rowsAffected(t, res))
	require.EqualValues(t, 99, skuStock(t, db, 2))
	require.EqualValues(t, 5, skuStock(t, db, 4)) // goods 9 is deleted, must be excluded by sd
}
