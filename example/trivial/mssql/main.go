package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/mssqldialect"
	"github.com/TommyLeng/bun/extra/bundebug"
	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	ctx := context.Background()

	// Open a MSSQL database.
	sqldb, err := sql.Open("sqlserver", "sqlserver://sa:passWORD1@localhost:1433?database=test")
	if err != nil {
		panic(err)
	}

	// Create a Bun db on top of it.
	db := bun.NewDB(sqldb, mssqldialect.New())

	// Print all queries to stdout.
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	var rnd float64

	// Select a random number.
	if err := db.NewSelect().ColumnExpr("rand()").Scan(ctx, &rnd); err != nil {
		panic(err)
	}

	fmt.Println(rnd)
}
