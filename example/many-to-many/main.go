package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"
)

type Order struct {
	ID    int64
	Items []Item `bun:"m2m:order_to_items"`
}

type Item struct {
	ID int64
}

type OrderToItem struct {
	OrderID int64  `bun:",pk"`
	Order   *Order `bun:"rel:belongs-to"`
	ItemID  int64  `bun:",pk"`
	Item    *Item  `bun:"rel:belongs-to"`
}

func main() {
	ctx := context.Background()

	sqldb, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()

	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))

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
	// Output: Order 1 Items 1 2
	// Order 1 Items 2 1
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
