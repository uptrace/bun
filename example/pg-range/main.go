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
	Room       int
	During     pgdialect.Range[time.Time]      `bun:"type:tsrange,default:'(,]'"`
	Durings    pgdialect.MultiRange[time.Time] `bun:"type:tsmultirange,multirange,default:'{}'"`
	DuringDate pgdialect.Range[time.Time]      `bun:"type:daterange,default:'(,]'"`
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

	{
		now := time.Now()
		st, end := now.Add(time.Hour), now.AddDate(0, 1, 1)
		reservations := []Reservation{
			{Room: 1, DuringDate: pgdialect.NewRange(st, end)},
			{Room: 2, During: pgdialect.NewRange(st, end)},
			// left is unbound
			{Room: 3, During: pgdialect.NewRange(
				time.Time{}, end,
				pgdialect.WithLowerBound[time.Time](pgdialect.RangeBoundUnbound))},
			// left & right is unbound
			{Room: 4, During: pgdialect.NewRange[time.Time](
				time.Time{}, time.Time{},
				pgdialect.WithLowerBound[time.Time](pgdialect.RangeBoundUnbound),
				pgdialect.WithUpperBound[time.Time](pgdialect.RangeBoundUnbound),
			)},
			// empty
			{Room: 5, During: pgdialect.NewRangeEmpty[time.Time]()},

			{Room: 10, Durings: pgdialect.MultiRange[time.Time]{
				pgdialect.NewRange(st, end),
				pgdialect.NewRange(st.Add(time.Hour*24), end.Add(time.Hour*24)),
			}},
		}

		if _, err := db.NewInsert().Model(&reservations).Exec(ctx); err != nil {
			panic(err)
		}
	}

	reservations := []Reservation{}
	if err := db.NewSelect().Model(&reservations).Order("room").Scan(ctx); err != nil {
		panic(err)
	}
	for _, m := range reservations {
		fmt.Println(m.Room, m.During, m.Durings)
	}
}
