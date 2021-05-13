package pgdriver

import (
	"errors"
	"fmt"
	"net/url"
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
		opts = append(opts, WithAddr(u.Host))
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

	if appName, ok := query["application_name"]; ok {
		opts = append(opts, WithApplicationName(appName[0]))
	}
	delete(query, "application_name")

	for key := range query {
		return nil, fmt.Errorf("pgdialect: unsupported option=%q", key)
	}

	return opts, nil
}
