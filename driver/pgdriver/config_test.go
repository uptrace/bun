package pgdriver_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"
)

func TestParseDSN(t *testing.T) {
	c := pgdriver.NewConnector(
		pgdriver.WithDSN("postgres://postgres:1@localhost:5432/testDatabase?sslmode=disable"),
	)

	cfg := c.Config()
	cfg.Dialer = nil

	require.Equal(t, &pgdriver.Config{
		Network:      "tcp",
		Addr:         "localhost:5432",
		User:         "postgres",
		Password:     "1",
		Database:     "testDatabase",
		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	}, cfg)
}
