package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type Reservation struct {
	Room   int
	During pgdialect.Range[time.Time] `bun:"type:tsrange"`
}

func main() {
	ctx := context.Background()

	dsn := "postgres://postgres:@localhost:5432/postgres?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	if err := db.ResetModel(ctx, (*Reservation)(nil)); err != nil {
		panic(err)
	}

	now := time.Now()
	st, end := now.Add(time.Hour), now.Add(time.Hour*2)
	reservations := []Reservation{
		{Room: 1, During: pgdialect.NewRange(&st, &end)},
		{Room: 2, During: pgdialect.NewRange(&st, &end)},
		{Room: 3, During: pgdialect.NewRange(nil, &end)},
		{Room: 4, During: pgdialect.NewRange[time.Time](nil, nil)},
		{Room: 5, During: pgdialect.NewEmptyRange[time.Time]()},
	}

	if _, err := db.NewInsert().Model(&reservations).Exec(ctx); err != nil {
		panic(err)
	}

	reservations = reservations[:0]
	if err := db.NewSelect().Model(&reservations).Scan(ctx); err != nil {
		panic(err)
	}
	for _, m := range reservations {
		fmt.Println(m.Room, m.During)
	}
}
