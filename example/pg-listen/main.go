package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/pgdialect"
	"github.com/TommyLeng/bun/driver/pgdriver"
	"github.com/TommyLeng/bun/extra/bundebug"
)

func main() {
	ctx := context.Background()

	dsn := "postgres://postgres:@localhost:5432/postgres?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	ln := pgdriver.NewListener(db)

	go func() {
		for i := 0; i < 3; i++ {
			payload := time.Now().Format(time.RFC3339)
			if err := pgdriver.Notify(ctx, db, "mychan1", payload); err != nil {
				panic(err)
			}
			time.Sleep(time.Second)
		}
		_ = ln.Close()
	}()

	if err := ln.Listen(ctx, "mychan1", "mychan2"); err != nil {
		panic(err)
	}

	for notif := range ln.Channel() {
		fmt.Println(notif.Channel, notif.Payload)
	}
}
