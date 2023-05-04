package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/sqlitedialect"
	"github.com/TommyLeng/bun/driver/sqliteshim"
)

type Item struct {
	ID int64 `bun:",pk,autoincrement"`
}

func main() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()

	q := db.NewSelect().Model((*Item)(nil)).Where("id > ?", 0).Where("id < ?", 10)

	fmt.Println(GetWhereFields(q.String()))
}

func GetWhereFields(query string) []string {
	q := strings.Split(query, "WHERE ")
	if len(q) == 1 {
		return nil
	}

	whereFields := strings.Split(q[1], " AND ")

	fields := make([]string, len(whereFields))

	for i := range whereFields {
		fields[i] = strings.TrimFunc(whereFields[i], func(r rune) bool {
			return r == '(' || r == ')'
		})
	}

	return fields
}
