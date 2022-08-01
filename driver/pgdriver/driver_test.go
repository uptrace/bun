package pgdriver_test

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/driver/pgdriver"
)

func BenchmarkExec(b *testing.B) {
	db, err := sql.Open("pg", dsn())
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := db.Exec("SELECT 1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSQLOpen(t *testing.T) {
	db, err := sql.Open("pg", dsn())
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	var str string
	err = db.QueryRow("SELECT $1", "hello").Scan(&str)
	require.NoError(t, err)
	require.Equal(t, "hello", str)

	err = db.Close()
	require.NoError(t, err)
}

func TestConnector(t *testing.T) {
	db := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn())))

	err := db.Ping()
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}

func TestConnector_WithResetSessionFunc(t *testing.T) {
	var resetCalled int

	db := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn()),
		pgdriver.WithResetSessionFunc(func(context.Context, *pgdriver.Conn) error {
			resetCalled++
			return nil
		}),
	))

	db.SetMaxOpenConns(1)

	for i := 0; i < 3; i++ {
		err := db.Ping()
		require.NoError(t, err)
	}

	require.Equal(t, 2, resetCalled)

	err := db.Close()
	require.NoError(t, err)
}

func TestStmtSelect(t *testing.T) {
	ctx := context.Background()
	db := sqlDB()

	stmt, err := db.Prepare("SELECT $1")
	require.NoError(t, err)

	res, err := stmt.Exec("hello")
	require.NoError(t, err)

	n, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	tests := []struct {
		s      string
		wanted string
	}{
		{s: "hello", wanted: "hello"},
		{s: "hell\000o", wanted: "hello"},
	}

	for _, test := range tests {
		var str string
		err = stmt.QueryRowContext(ctx, test.s).Scan(&str)
		require.NoError(t, err)
		require.Equal(t, test.wanted, str)
	}

	err = stmt.Close()
	require.NoError(t, err)

	_, err = stmt.Exec("hello")
	require.Error(t, err)
	require.Equal(t, "sql: statement is closed", err.Error())

	err = db.Ping()
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}

func TestStmtNoParams(t *testing.T) {
	db := sqlDB()
	defer db.Close()

	stmt, err := db.Prepare("SELECT 1")
	require.NoError(t, err)

	res, err := stmt.Exec()
	require.NoError(t, err)

	n, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	err = stmt.Close()
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}

func TestStmtConcurrency(t *testing.T) {
	db := sqlDB()
	defer db.Close()

	var wg sync.WaitGroup
	var stopped uint32

	wg.Add(1)
	go func() {
		defer wg.Done()

		for atomic.LoadUint32(&stopped) == 0 {
			stmt1, err := db.Prepare("SELECT $1")
			require.NoError(t, err)

			var n1 int
			err = stmt1.QueryRow(123).Scan(&n1)
			require.NoError(t, err)
			require.Equal(t, 123, n1)

			err = stmt1.Close()
			require.NoError(t, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for atomic.LoadUint32(&stopped) == 0 {
			stmt2, err := db.Prepare("SELECT $1, $2")
			require.NoError(t, err)

			var n1, n2 int
			err = stmt2.QueryRow(456, 789).Scan(&n1, &n2)
			require.NoError(t, err)
			require.Equal(t, 456, n1)
			require.Equal(t, 789, n2)

			err = stmt2.Close()
			require.NoError(t, err)
		}
	}()

	time.Sleep(time.Second)
	atomic.StoreUint32(&stopped, 1)
	wg.Wait()
}

func TestCancel(t *testing.T) {
	db := sqlDB()
	defer db.Close()

	var wg sync.WaitGroup
	var stopped uint32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for atomic.LoadUint32(&stopped) == 0 {
				ctx := context.Background()
				ctx, cancel := context.WithCancel(ctx)
				go func() {
					time.Sleep(10 * time.Millisecond) // same as pg_sleep
					cancel()
				}()
				_, _ = db.ExecContext(ctx, "select pg_sleep(0.01)")
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for atomic.LoadUint32(&stopped) == 0 {
			ctx := context.Background()
			_, err := db.ExecContext(ctx, "select pg_sleep(0.01)")
			require.NoError(t, err)
		}
	}()

	time.Sleep(3 * time.Second)
	atomic.StoreUint32(&stopped, 1)
	wg.Wait()
}

func TestFloat64(t *testing.T) {
	db := sqlDB()
	defer db.Close()

	var f float64
	err := db.QueryRow("SELECT 1.1::float AS f").Scan(&f)
	require.NoError(t, err)
	require.Equal(t, 1.1, f)
}

func TestConnParams(t *testing.T) {
	db := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn()),
		pgdriver.WithConnParams(map[string]interface{}{
			"search_path": "foo",
		}),
	))
	defer db.Close()

	var searchPath string
	err := db.QueryRow("SHOW search_path").Scan(&searchPath)
	require.NoError(t, err)
	require.Equal(t, "foo", searchPath)
}

func TestStatementTimeout(t *testing.T) {
	ctx := context.Background()

	db := sqlDB()
	defer db.Close()

	cn, err := db.Conn(ctx)
	require.NoError(t, err)

	_, err = cn.ExecContext(ctx, "SET statement_timeout = 100")
	require.NoError(t, err)

	_, err = cn.ExecContext(ctx, "SELECT pg_sleep(1)")
	require.Error(t, err)

	pgerr, ok := err.(pgdriver.Error)
	require.True(t, ok)
	require.True(t, pgerr.StatementTimeout())
}

func sqlDB() *sql.DB {
	db, err := sql.Open("pg", dsn())
	if err != nil {
		panic(err)
	}
	return db
}

func dsn() string {
	dsn := os.Getenv("PG")
	if dsn == "" {
		dsn = "postgres://postgres:@localhost:5432/test?sslmode=disable"
	}
	return dsn
}
