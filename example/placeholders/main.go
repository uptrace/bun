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

type User struct {
	ID     int64 `bun:",pk,autoincrement"`
	Name   string
	Emails []string
}

func main() {
	ctx := context.Background()

	sqlite, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqlite.SetMaxOpenConns(1)

	db := bun.NewDB(sqlite, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	var tableName, tableAlias, pks, tablePKs, columns, tableColumns string

	if err := db.NewSelect().Model((*User)(nil)).
		ColumnExpr("'?TableName'").
		ColumnExpr("'?TableAlias'").
		ColumnExpr("'?PKs'").
		ColumnExpr("'?TablePKs'").
		ColumnExpr("'?Columns'").
		ColumnExpr("'?TableColumns'").
		ModelTableExpr("").
		Scan(ctx, &tableName, &tableAlias, &pks, &tablePKs, &columns, &tableColumns); err != nil {
		panic(err)
	}

	fmt.Println("tableName", tableName)
	fmt.Println("tableAlias", tableAlias)
	fmt.Println("pks", pks)
	fmt.Println("tablePKs", tablePKs)
	fmt.Println("columns", columns)
	fmt.Println("tableColumns", tableColumns)
}
