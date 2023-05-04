package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/sqlitedialect"
	"github.com/TommyLeng/bun/driver/sqliteshim"
	"github.com/TommyLeng/bun/extra/bundebug"
)

type Order struct {
	ID int64 `bun:",pk,autoincrement"`
	// Order and Item in join:Order=Item are fields in OrderToItem model
	Items []Item `bun:"m2m:order_to_items,join:Order=Item"`
}

type Item struct {
	ID int64 `bun:",pk,autoincrement"`
	// Order and Item in join:Order=Item are fields in OrderToItem model
	Orders []Order `bun:"m2m:order_to_items,join:Item=Order"`
}

type OrderToItem struct {
	OrderID int64  `bun:",pk"`
	Order   *Order `bun:"rel:belongs-to,join:order_id=id"`
	ItemID  int64  `bun:",pk"`
	Item    *Item  `bun:"rel:belongs-to,join:item_id=id"`
}

func main() {
	ctx := context.Background()

	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()

	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// Register many to many model so bun can better recognize m2m relation.
	// This should be done before you use the model for the first time.
	db.RegisterModel((*OrderToItem)(nil))

	if err := createSchema(ctx, db); err != nil {
		panic(err)
	}

	order := new(Order)
	if err := db.NewSelect().
		Model(order).
		Relation("Items").
		Order("order.id ASC").
		Limit(1).
		Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Order", order.ID, "Items", order.Items[0].ID, order.Items[1].ID)
	fmt.Println()

	order = new(Order)
	if err := db.NewSelect().
		Model(order).
		Relation("Items", func(q *bun.SelectQuery) *bun.SelectQuery {
			q = q.OrderExpr("item.id DESC")
			return q
		}).
		Limit(1).
		Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Order", order.ID, "Items", order.Items[0].ID, order.Items[1].ID)
	fmt.Println()

	item := new(Item)
	if err := db.NewSelect().
		Model(item).
		Relation("Orders").
		Order("item.id ASC").
		Limit(1).
		Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Item", item.ID, "Order", item.Orders[0].ID)
}

func createSchema(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*Order)(nil),
		(*Item)(nil),
		(*OrderToItem)(nil),
	}
	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).Exec(ctx); err != nil {
			return err
		}
	}

	values := []interface{}{
		&Item{ID: 1},
		&Item{ID: 2},
		&Order{ID: 1},
		&OrderToItem{OrderID: 1, ItemID: 1},
		&OrderToItem{OrderID: 1, ItemID: 2},
	}
	for _, value := range values {
		if _, err := db.NewInsert().Model(value).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
