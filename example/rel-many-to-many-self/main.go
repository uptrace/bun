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

type Item struct {
	ID       int64  `bun:",pk,autoincrement"`
	Children []Item `bun:"m2m:item_to_items,join:Item=Child"`
}

type ItemToItem struct {
	ItemID  int64 `bun:",pk"`
	Item    *Item `bun:"rel:belongs-to,join:item_id=id"`
	ChildID int64 `bun:",pk"`
	Child   *Item `bun:"rel:belongs-to,join:child_id=id"`
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
	db.RegisterModel((*ItemToItem)(nil))

	if err := createSchema(ctx, db); err != nil {
		panic(err)
	}

	item1 := new(Item)
	if err := db.NewSelect().
		Model(item1).
		Relation("Children").
		Order("item.id ASC").
		Where("id = 1").
		Scan(ctx); err != nil {
		panic(err)
	}

	item2 := new(Item)
	if err := db.NewSelect().
		Model(item2).
		Relation("Children").
		Order("item.id ASC").
		Where("id = 2").
		Scan(ctx); err != nil {
		panic(err)
	}

	fmt.Println("item1", item1.ID, "children", item1.Children[0].ID, item1.Children[1].ID)
	fmt.Println("item2", item2.ID, "children", item2.Children[0].ID, item2.Children[1].ID)
}

func createSchema(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*Item)(nil),
		(*ItemToItem)(nil),
	}
	for _, model := range models {
		if err := db.ResetModel(ctx, model); err != nil {
			return err
		}
	}

	values := []interface{}{
		&Item{ID: 1},
		&Item{ID: 2},
		&Item{ID: 3},
		&Item{ID: 4},
		&ItemToItem{ItemID: 1, ChildID: 2},
		&ItemToItem{ItemID: 1, ChildID: 3},
		&ItemToItem{ItemID: 2, ChildID: 3},
		&ItemToItem{ItemID: 2, ChildID: 4},
	}
	for _, value := range values {
		if _, err := db.NewInsert().Model(value).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
