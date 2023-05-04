package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/TommyLeng/bun"
	"github.com/TommyLeng/bun/dialect/pgdialect"
	"github.com/TommyLeng/bun/driver/pgdriver"
	"github.com/TommyLeng/bun/extra/bunotel"
	"github.com/brianvoe/gofakeit/v5"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("github.com/TommyLeng/bun/example/opentelemetry")

func main() {
	ctx := context.Background()

	uptrace.ConfigureOpentelemetry(
		// copy your project DSN here or use UPTRACE_DSN env var
		// uptrace.WithDSN("http://project2_secret_token@localhost:14317/2"),

		uptrace.WithServiceName("myservice"),
		uptrace.WithServiceVersion("v1.0.0"),
	)
	defer uptrace.Shutdown(ctx)

	dsn := "postgres://uptrace:uptrace@localhost:5432/uptrace?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bunotel.NewQueryHook(
		bunotel.WithDBName("uptrace"),
		bunotel.WithFormattedQueries(true),
	))
	// db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	if err := db.ResetModel(ctx, (*TestModel)(nil)); err != nil {
		panic(err)
	}

	for i := 0; i < 1e6; i++ {
		ctx, rootSpan := tracer.Start(ctx, "handleRequest")

		if err := handleRequest(ctx, db); err != nil {
			rootSpan.RecordError(err)
			rootSpan.SetStatus(codes.Error, err.Error())
		}

		rootSpan.End()

		if i == 0 {
			fmt.Printf("view trace: %s\n", uptrace.TraceURL(rootSpan))
		}

		time.Sleep(time.Second)
	}
}

type TestModel struct {
	ID   int64 `bun:",pk,autoincrement"`
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
