package pgdriver

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type DriverOption func(*driverConnector)

func WithAddr(addr string) DriverOption {
	return func(d *driverConnector) {
		d.addr = addr
	}
}

func WithUser(user string) DriverOption {
	return func(d *driverConnector) {
		d.user = user
	}
}

func WithPassword(password string) DriverOption {
	return func(d *driverConnector) {
		d.password = password
	}
}

func WithDatabase(database string) DriverOption {
	return func(d *driverConnector) {
		d.database = database
	}
}

func WithApplicationName(appName string) DriverOption {
	return func(d *driverConnector) {
		d.appName = appName
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
