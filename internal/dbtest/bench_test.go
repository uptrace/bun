package dbtest_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

type Bench struct {
	ID        int64 `bun:",pk,autoincrement"`
	Name      string
	CreatedAt time.Time
}

func BenchmarkSelectOne(b *testing.B) {
	benchEachDB(b, benchmarkSelectOne)
}

func benchmarkSelectOne(b *testing.B, db *bun.DB) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bench := new(Bench)
			err := db.NewSelect().Model(bench).Where("id = ?", 1).Scan(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSelectSlice(b *testing.B) {
	benchEachDB(b, benchmarkSelectSlice)
}

func benchmarkSelectSlice(b *testing.B, db *bun.DB) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var bs []Bench
			err := db.NewSelect().Model(&bs).Limit(100).Scan(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSelectError(b *testing.B) {
	benchEachDB(b, benchmarkSelectError)
}

func benchmarkSelectError(b *testing.B, db *bun.DB) {
	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			if i%2 == 0 {
				_, err := db.Exec("SELECT * FROM unknown_table")
				require.Error(b, err)
			} else {
				var num int
				err := db.NewSelect().ColumnExpr("123").Scan(ctx, &num)
				require.NoError(b, err)
				require.Equal(b, 123, num)
			}
			i++
		}
	})
}

func benchEachDB(b *testing.B, f func(b *testing.B, db *bun.DB)) {
	for name, newDB := range allDBs {
		b.Run(name, func(b *testing.B) {
			db := newDB(b)
			db.SetMaxOpenConns(64)
			db.SetMaxIdleConns(64)

			err := resetBenchSchema(b, db)
			require.NoError(b, err)

			b.ResetTimer()
			f(b, db)
		})
	}
}

func resetBenchSchema(tb testing.TB, db *bun.DB) error {
	mustResetModel(tb, ctx, db, (*Bench)(nil))

	for i := 0; i < 1000; i++ {
		bench := &Bench{
			Name:      gofakeit.Name(),
			CreatedAt: time.Now(),
		}
		if _, err := db.NewInsert().Model(bench).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
