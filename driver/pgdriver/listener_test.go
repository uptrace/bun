package pgdriver_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func TestListener(t *testing.T) {
	ctx := context.Background()

	db := db()
	defer db.Close()

	ln := pgdriver.NewListener(db)

	_, _, err := ln.ReceiveTimeout(ctx, 200*time.Millisecond)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout")

	err = ln.Listen(ctx, "test_channel")
	require.NoError(t, err)

	_, err = db.Exec("NOTIFY test_channel")
	require.NoError(t, err)

	channel, payload, err := ln.Receive(ctx)
	require.NoError(t, err)
	require.Equal(t, "test_channel", channel)
	require.Equal(t, "", payload)

	_, err = db.Exec("NOTIFY test_channel, ?", "test_payload")
	require.NoError(t, err)

	channel, payload, err = ln.Receive(ctx)
	require.NoError(t, err)
	require.Equal(t, "test_channel", channel)
	require.Equal(t, "test_payload", payload)

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = ln.Close()
	}()

	_, _, err = ln.Receive(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func db() *bun.DB {
	dsn := os.Getenv("PG")
	if dsn == "" {
		dsn = "postgres://postgres:@localhost:5432/test?sslmode=disable"
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	return bun.Open(sqldb, pgdialect.New())
}
