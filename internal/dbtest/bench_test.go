package dbtest_test

import (
	"sync"
	"testing"
	"time"

	"github.com/uptrace/bun"
)

type Bench struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

func BenchmarkSelect(b *testing.B) {
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

	if _, err := db.NewDropTable().Model(&Bench{}).IfExists().Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().Model(&Bench{}).Exec(ctx); err != nil {
		return err
	}

	bench := &Bench{
		ID:        1,
		Name:      "John Doe",
		CreatedAt: time.Now(),
	}
	if _, err := db.NewInsert().Model(bench).Exec(ctx); err != nil {
		return err
	}

	return nil
}
