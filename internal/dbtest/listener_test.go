package dbtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TommyLeng/bun/driver/pgdriver"
)

func TestListenerReceive(t *testing.T) {
	ctx := context.Background()

	db := pg(t)

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

func TestListenerChannel(t *testing.T) {
	ctx := context.Background()

	db := pg(t)

	ln := pgdriver.NewListener(db)
	ch := ln.Channel()

	err := ln.Listen(ctx, "test_channel")
	require.NoError(t, err)

	for _, msg := range []string{"foo", "bar"} {
		_, err = db.Exec("NOTIFY test_channel, ?", msg)
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		_, ok := <-ch
		return ok
	}, 3*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		_, ok := <-ch
		return ok
	}, 3*time.Second, 100*time.Millisecond)

	err = ln.Close()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, ok := <-ch
		return !ok
	}, 3*time.Second, 100*time.Millisecond)
}
