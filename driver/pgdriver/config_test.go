package pgdriver_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"
)

func TestParseDSN(t *testing.T) {
	type Test struct {
		dsn string
		cfg *pgdriver.Config
	}

	tests := []Test{
		{
			dsn: "postgres://postgres:1@localhost:5432/testDatabase?sslmode=disable",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "localhost:5432",
				User:         "postgres",
				Password:     "1",
				Database:     "testDatabase",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			dsn: "postgres://postgres:1@localhost:5432/testDatabase?sslmode=disable&dial_timeout=1&read_timeout=2s&write_timeout=3",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "localhost:5432",
				User:         "postgres",
				Password:     "1",
				Database:     "testDatabase",
				DialTimeout:  1 * time.Second,
				ReadTimeout:  2 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
		{
			dsn: "postgres://postgres:password@app.xxx.us-east-1.rds.amazonaws.com:5432/test?sslmode=disable",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "app.xxx.us-east-1.rds.amazonaws.com:5432",
				User:         "postgres",
				Password:     "password",
				Database:     "test",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
	}

	for _, test := range tests {
		c := pgdriver.NewConnector(pgdriver.WithDSN(test.dsn))

		cfg := c.Config()
		cfg.Dialer = nil

		require.Equal(t, test.cfg, cfg)
	}
}
