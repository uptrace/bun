package main

import (
	"context"
	"database/sql"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/opentelemetry-go-extra/otelplay"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("bunexample")

func main() {
	ctx := context.Background()

	shutdown := otelplay.ConfigureOpentelemetry(ctx)
	defer shutdown()

	sqlite, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqlite.SetMaxOpenConns(1)

	db := bun.NewDB(sqlite, sqlitedialect.New())
	db.AddQueryHook(bunotel.NewQueryHook())
	// db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	if _, err := db.NewCreateTable().Model((*TestModel)(nil)).Exec(ctx); err != nil {
		panic(err)
	}

	ctx, span := tracer.Start(ctx, "handleRequest")
	defer span.End()

	if err := handleRequest(ctx, db); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	otelplay.PrintTraceID(ctx)
}

type TestModel struct {
	ID   int64
	Name string
}

func handleRequest(ctx context.Context, db *bun.DB) error {
	model := &TestModel{
		Name: gofakeit.Name(),
	}
	if _, err := db.NewInsert().Model(model).Exec(ctx); err != nil {
		return err
	}

	// Check that data can be selected without any errors.
	if err := db.NewSelect().Model(model).WherePK().Scan(ctx); err != nil {
		return err
	}

	return nil
}
