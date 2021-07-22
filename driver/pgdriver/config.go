package pgdriver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Network type, either tcp or unix.
	// Default is tcp.
	Network string
	// TCP host:port or Unix socket depending on Network.
	Addr string
	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout time.Duration
	// Dialer creates new network connection and has priority over
	// Network and Addr options.
	Dialer func(ctx context.Context, network, addr string) (net.Conn, error)

	// TLS config for secure connections.
	TLSConfig *tls.Config

	User     string
	Password string
	Database string
	AppName  string

	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking.
	ReadTimeout time.Duration
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
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

func WithTLSConfig(cfg *tls.Config) DriverOption {
	return func(d *driverConnector) {
		d.cfg.TLSConfig = cfg
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

	if sslMode := query.Get("sslmode"); sslMode != "" {
		switch sslMode {
		case "verify-ca", "verify-full":
			opts = append(opts, WithTLSConfig(new(tls.Config)))
		case "allow", "prefer", "require":
			opts = append(opts, WithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
		case "disable":
			// no TLS config
		default:
			return nil, fmt.Errorf("pgdriver: sslmode '%s' is not supported", sslMode)
		}
	} else {
		opts = append(opts, WithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	}
	delete(query, "sslmode")

	for key := range query {
		return nil, fmt.Errorf("pgdriver: unsupported option=%q", key)
	}

	return opts, nil
}

func env(key, defValue string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return defValue
}

// verify is a method to make sure if the config is legitimate
// in the case it detects any errors, it returns with a non-nil error
// it can be extended to check other parameters
func (c *Config) verify() error {
	if c.User == "" {
		return errors.New(
			`pgdriver: username is required. 
		 	 Check your config and make sure User is not left blank.`,
		)
	}
	return nil
}
