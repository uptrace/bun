package pgdriver_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/driver/pgdriver"
)

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

func TestStmt(t *testing.T) {
	ctx := context.Background()
	db := sqlDB()

	stmt, err := db.Prepare("SELECT $1")
	require.NoError(t, err)

	res, err := stmt.ExecContext(ctx, "hello")
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

	_, err = stmt.ExecContext(ctx, "hello")
	require.Error(t, err)
	require.Equal(t, "sql: statement is closed", err.Error())

	err = db.Ping()
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
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
		dsn = "postgres://postgres:@localhost:5432/test"
	}
	return dsn
}
