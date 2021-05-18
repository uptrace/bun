package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Network     string
	Addr        string
	DialTimeout time.Duration
	Dialer      func(ctx context.Context, network, addr string) (net.Conn, error)

	User     string
	Password string
	Database string
	AppName  string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func newDefaultConfig() Config {
	host := env("PGHOST", "localhost")
	port := env("PGPORT", "5432")

	return Config{
		Network:     "tcp",
		Addr:        net.JoinHostPort(host, port),
		DialTimeout: 5 * time.Second,

		User:     env("PGUSER", "postgres"),
		Database: env("PGDATABASE", "postgres"),

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

type DriverOption func(*driverConnector)

func WithAddr(addr string) DriverOption {
	return func(d *driverConnector) {
		d.cfg.Addr = addr
	}
}

func WithUser(user string) DriverOption {
	return func(d *driverConnector) {
		d.cfg.User = user
	}
}

func WithPassword(password string) DriverOption {
	return func(d *driverConnector) {
		d.cfg.Password = password
	}
}

func WithDatabase(database string) DriverOption {
	return func(d *driverConnector) {
		d.cfg.Database = database
	}
}

func WithApplicationName(appName string) DriverOption {
	return func(d *driverConnector) {
		d.cfg.AppName = appName
	}
}

func WithTimeout(timeout time.Duration) DriverOption {
	return func(d *driverConnector) {
		d.cfg.DialTimeout = timeout
		d.cfg.ReadTimeout = timeout
		d.cfg.WriteTimeout = timeout
	}
}

func WithDialTimeout(dialTimeout time.Duration) DriverOption {
	return func(d *driverConnector) {
		d.cfg.DialTimeout = dialTimeout
	}
}

func WithReadTimeout(readTimeout time.Duration) DriverOption {
	return func(d *driverConnector) {
		d.cfg.ReadTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) DriverOption {
	return func(d *driverConnector) {
		d.cfg.WriteTimeout = writeTimeout
	}
}

func WithDSN(dsn string) DriverOption {
	return func(d *driverConnector) {
		opts, err := parseDSN(dsn)
		if err != nil {
			panic(err)
		}
		for _, opt := range opts {
			opt(d)
		}
	}
}

func parseDSN(dsn string) ([]DriverOption, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, errors.New("pgdriver: invalid scheme: " + u.Scheme)
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	var opts []DriverOption

	if u.Host != "" {
		addr := u.Host
		if !strings.Contains(addr, ":") {
			addr += ":5432"
		}
		opts = append(opts, WithAddr(addr))
	}
	if u.User != nil {
		opts = append(opts, WithUser(u.User.Username()))
		if password, ok := u.User.Password(); ok {
			opts = append(opts, WithPassword(password))
		}
	}
	if len(u.Path) > 1 {
		opts = append(opts, WithDatabase(u.Path[1:]))
	}

	if appName := query.Get("application_name"); appName != "" {
		opts = append(opts, WithApplicationName(appName))
	}
	delete(query, "application_name")

	if sslMode := query.Get("sslmode"); sslMode != "" { //nolint:staticcheck
		// TODO: implement
	}
	delete(query, "sslmode")

	for key := range query {
		return nil, fmt.Errorf("pgdialect: unsupported option=%q", key)
	}

	return opts, nil
}

func env(key, defValue string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return defValue
}
