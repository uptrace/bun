package pgdriver_test

import (
	"fmt"
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
			dsn: "postgres://user:password@localhost:5432/testDatabase?sslmode=disable",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "localhost:5432",
				User:         "user",
				Password:     "password",
				Database:     "testDatabase",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			dsn: "postgres://user:password@localhost:5432/testDatabase?sslmode=disable&dial_timeout=1&read_timeout=2s&write_timeout=3",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "localhost:5432",
				User:         "user",
				Password:     "password",
				Database:     "testDatabase",
				DialTimeout:  1 * time.Second,
				ReadTimeout:  2 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
		{
			dsn: "postgres://user:password@localhost:5432/testDatabase?search_path=foo",
			cfg: &pgdriver.Config{
				Network:  "tcp",
				Addr:     "localhost:5432",
				User:     "user",
				Password: "password",
				Database: "testDatabase",
				ConnParams: map[string]interface{}{
					"search_path": "foo",
				},
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			dsn: "postgres://user:password@app.xxx.us-east-1.rds.amazonaws.com:5432/test?sslmode=disable",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "app.xxx.us-east-1.rds.amazonaws.com:5432",
				User:         "user",
				Password:     "password",
				Database:     "test",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			dsn: "postgres://user:password@/dbname?host=/var/run/postgresql/.s.PGSQL.5432",
			cfg: &pgdriver.Config{
				Network:      "unix",
				Addr:         "/var/run/postgresql/.s.PGSQL.5432",
				User:         "user",
				Password:     "password",
				Database:     "dbname",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			dsn: "unix://user:pass@dbname/var/run/postgresql/.s.PGSQL.5432",
			cfg: &pgdriver.Config{
				Network:      "unix",
				Addr:         "/var/run/postgresql/.s.PGSQL.5432",
				User:         "user",
				Password:     "pass",
				Database:     "dbname",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			dsn: "postgres://user:password@localhost:5432/testDatabase?connect_timeout=3",
			cfg: &pgdriver.Config{
				Network:      "tcp",
				Addr:         "localhost:5432",
				User:         "user",
				Password:     "password",
				Database:     "testDatabase",
				DialTimeout:  3 * time.Second,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := pgdriver.NewConnector(pgdriver.WithDSN(test.dsn))

			cfg := c.Config()
			cfg.Dialer = nil
			cfg.TLSConfig = nil

			require.Equal(t, test.cfg, cfg)
		})
	}
}
