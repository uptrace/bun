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

func main() {
	ctx := context.Background()

	dsn := "postgres://postgres:@localhost:5432/postgres?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	go func() {
		for tm := range time.Tick(time.Second) {
			if err := pgdriver.Notify(ctx, db, "mychan1", tm.Format(time.RFC3339)); err != nil {
				panic(err)
			}
		}
	}()

	ln := pgdriver.NewListener(db)
	if err := ln.Listen(ctx, "mychan1", "mychan2"); err != nil {
		panic(err)
	}

	for notif := range ln.Channel() {
		fmt.Println(notif.Channel, notif.Payload)
	}
}
