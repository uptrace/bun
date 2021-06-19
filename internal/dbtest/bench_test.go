package dbtest_test

import (
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

type Bench struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

func BenchmarkSelectOne(b *testing.B) {
	db := benchDB()

	b.ResetTimer()

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
	db := benchDB()

	b.ResetTimer()

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
	db := benchDB()

	b.ResetTimer()

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

var (
	dbOnce sync.Once
	db     *bun.DB
)

func benchDB() *bun.DB {
	dbOnce.Do(func() {
		db = pg()
		db.SetMaxOpenConns(64)
		db.SetMaxIdleConns(64)

		if err := resetBenchSchema(); err != nil {
			panic(err)
		}
	})
	return db
}

func resetBenchSchema() error {
	db := pg()
	defer db.Close()

	if err := db.ResetModel(ctx, (*Bench)(nil)); err != nil {
		return err
	}

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
