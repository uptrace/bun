package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dbfixture"
	"github.com/TommyLeng/bun/dialect/pgdialect"
	"github.com/TommyLeng/bun/driver/pgdriver"
	"github.com/TommyLeng/bun/extra/bundebug"
	"github.com/davecgh/go-spew/spew"
)

type Book struct {
	ID   uint64 `bun:",pk,autoincrement"`
	Name string
	Tags []string
}

var _ bun.BeforeCreateTableHook = (*Book)(nil)

func (*Book) BeforeCreateTable(ctx context.Context, q *bun.CreateTableQuery) error {
	q.ColumnExpr("tsv tsvector")
	return nil
}

var _ bun.AfterCreateTableHook = (*Book)(nil)

func (*Book) AfterCreateTable(ctx context.Context, query *bun.CreateTableQuery) error {
	_, err := query.DB().NewCreateIndex().
		Model((*Book)(nil)).
		Index("books_tsv_idx").
		Using("GIN").
		Column("tsv").
		Exec(ctx)
	return err
}

func main() {
	ctx := context.Background()

	dsn := "postgres://postgres:@localhost:5432/test?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// Register models for the fixture.
	db.RegisterModel((*Book)(nil))

	// Create tables and load initial data.
	fixture := dbfixture.New(db,
		dbfixture.WithRecreateTables(),
		dbfixture.WithBeforeInsert(beforeInsertFixture))
	if err := fixture.Load(ctx, os.DirFS("."), "fixture.yml"); err != nil {
		panic(err)
	}

	facets, err := selectFacets(ctx, db)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n\nall facets:\n\n")
	spew.Dump(facets)

	facets, err = selectFacets(ctx, db, "moods:mysterious")
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n\nmoods:mysterious facets:\n\n")
	spew.Dump(facets)
}

type Facet struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Count uint32 `json:"count"`
}

type FacetMap map[string][]*Facet

func selectFacets(ctx context.Context, db *bun.DB, tags ...string) (FacetMap, error) {
	q := db.NewSelect().
		Column("tsv").
		Model((*Book)(nil))

	for _, tag := range tags {
		tag = strings.ReplaceAll(tag, ":", "\\:")
		q = q.Where("tsv @@ ?::tsquery", tag)
	}

	var facets []*Facet

	if err := db.NewSelect().
		ColumnExpr("split_part(word, ':', 1) AS key").
		ColumnExpr("split_part(word, ':', 2) AS value").
		ColumnExpr("ndoc AS count").
		ColumnExpr("row_number() OVER ("+
			"PARTITION BY split_part(word, ':', 1) "+
			"ORDER BY ndoc DESC"+
			") AS _rank").
		TableExpr("ts_stat($$ ? $$)", q).
		OrderExpr("_rank DESC").
		Scan(ctx, &facets); err != nil {
		return nil, err
	}

	m := make(FacetMap)

	for _, facet := range facets {
		m[facet.Key] = append(m[facet.Key], facet)
	}

	return m, nil
}

func beforeInsertFixture(ctx context.Context, data *dbfixture.BeforeInsertData) error {
	book := data.Model.(*Book)
	data.Query.Value("tsv", "array_to_tsvector(?)", pgdialect.Array(book.Tags))
	return nil
}
