package pgdriver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
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

	// Timeout for socket reads. If reached, commands fail with a timeout instead of blocking.
	ReadTimeout time.Duration
	// Timeout for socket writes. If reached, commands fail with a timeout instead of blocking.
	WriteTimeout time.Duration
}

func newDefaultConfig() *Config {
	host := env("PGHOST", "localhost")
	port := env("PGPORT", "5432")

	cfg := &Config{
		Network:     "tcp",
		Addr:        net.JoinHostPort(host, port),
		DialTimeout: 5 * time.Second,

		User:     env("PGUSER", "postgres"),
		Database: env("PGDATABASE", "postgres"),

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netDialer := &net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 5 * time.Minute,
		}
		return netDialer.DialContext(ctx, network, addr)
	}

	return cfg
}

type DriverOption func(*Connector)

func WithAddr(addr string) DriverOption {
	if addr == "" {
		panic("addr is empty")
	}
	return func(d *Connector) {
		d.cfg.Addr = addr
	}
}

func WithTLSConfig(cfg *tls.Config) DriverOption {
	return func(d *Connector) {
		d.cfg.TLSConfig = cfg
	}
}

func WithUser(user string) DriverOption {
	if user == "" {
		panic("user is empty")
	}
	return func(d *Connector) {
		d.cfg.User = user
	}
}

func WithPassword(password string) DriverOption {
	return func(d *Connector) {
		d.cfg.Password = password
	}
}

func WithDatabase(database string) DriverOption {
	if database == "" {
		panic("database is empty")
	}
	return func(d *Connector) {
		d.cfg.Database = database
	}
}

func WithApplicationName(appName string) DriverOption {
	return func(d *Connector) {
		d.cfg.AppName = appName
	}
}

func WithTimeout(timeout time.Duration) DriverOption {
	return func(d *Connector) {
		d.cfg.DialTimeout = timeout
		d.cfg.ReadTimeout = timeout
		d.cfg.WriteTimeout = timeout
	}
}

func WithDialTimeout(dialTimeout time.Duration) DriverOption {
	return func(d *Connector) {
		d.cfg.DialTimeout = dialTimeout
	}
}

func WithReadTimeout(readTimeout time.Duration) DriverOption {
	return func(d *Connector) {
		d.cfg.ReadTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) DriverOption {
	return func(d *Connector) {
		d.cfg.WriteTimeout = writeTimeout
	}
}

func WithDSN(dsn string) DriverOption {
	return func(d *Connector) {
		opts, err := parseDSN(dsn)
		if err != nil {
			panic(err)
		}
		for _, opt := range opts {
			opt(d)
		}
	}
}

func env(key, defValue string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return defValue
}

//------------------------------------------------------------------------------

func parseDSN(dsn string) ([]DriverOption, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, errors.New("pgdriver: invalid scheme: " + u.Scheme)
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

	q := queryOptions{q: u.Query()}

	if appName := q.string("application_name"); appName != "" {
		opts = append(opts, WithApplicationName(appName))
	}

	switch sslMode := q.string("sslmode"); sslMode {
	case "verify-ca", "verify-full":
		opts = append(opts, WithTLSConfig(new(tls.Config)))
	case "allow", "prefer", "require":
		opts = append(opts, WithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	case "disable", "":
		// no TLS config
	default:
		return nil, fmt.Errorf("pgdriver: sslmode '%s' is not supported", sslMode)
	}

	if d := q.duration("timeout"); d != 0 {
		opts = append(opts, WithTimeout(d))
	}
	if d := q.duration("dial_timeout"); d != 0 {
		opts = append(opts, WithDialTimeout(d))
	}
	if d := q.duration("read_timeout"); d != 0 {
		opts = append(opts, WithReadTimeout(d))
	}
	if d := q.duration("write_timeout"); d != 0 {
		opts = append(opts, WithWriteTimeout(d))
	}

	rem, err := q.remaining()
	if err != nil {
		return nil, q.err
	}
	if len(rem) > 0 {
		return nil, fmt.Errorf("pgdriver: unexpected option: %s", strings.Join(rem, ", "))
	}

	return opts, nil
}

// verify is a method to make sure if the config is legitimate
// in the case it detects any errors, it returns with a non-nil error
// it can be extended to check other parameters
func (c *Config) verify() error {
	if c.User == "" {
		return errors.New("pgdriver: User option is empty (to configure, use WithUser).")
	}
	return nil
}

type queryOptions struct {
	q   url.Values
	err error
}

func (o *queryOptions) string(name string) string {
	vs := o.q[name]
	if len(vs) == 0 {
		return ""
	}
	delete(o.q, name) // enable detection of unknown parameters
	return vs[len(vs)-1]
}

func (o *queryOptions) duration(name string) time.Duration {
	s := o.string(name)
	if s == "" {
		return 0
	}
	// try plain number first
	if i, err := strconv.Atoi(s); err == nil {
		if i <= 0 {
			// disable timeouts
			return -1
		}
		return time.Duration(i) * time.Second
	}
	dur, err := time.ParseDuration(s)
	if err == nil {
		return dur
	}
	if o.err == nil {
		o.err = fmt.Errorf("pgdriver: invalid %s duration: %w", name, err)
	}
	return 0
}

func (o *queryOptions) remaining() ([]string, error) {
	if o.err != nil {
		return nil, o.err
	}
	if len(o.q) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(o.q))
	for k := range o.q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}
