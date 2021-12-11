package pgdriver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
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
	// PostgreSQL session parameters updated with `SET` command when a connection is created.
	ConnParams map[string]interface{}

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
		TLSConfig:   &tls.Config{InsecureSkipVerify: true},

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

type Option func(cfg *Config)

// Deprecated. Use Option instead.
type DriverOption = Option

func WithNetwork(network string) Option {
	if network == "" {
		panic("network is empty")
	}
	return func(cfg *Config) {
		cfg.Network = network
	}
}

func WithAddr(addr string) Option {
	if addr == "" {
		panic("addr is empty")
	}
	return func(cfg *Config) {
		cfg.Addr = addr
	}
}

func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(cfg *Config) {
		cfg.TLSConfig = tlsConfig
	}
}

func WithInsecure(on bool) Option {
	return func(cfg *Config) {
		if on {
			cfg.TLSConfig = nil
		} else {
			cfg.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}
}

func WithUser(user string) Option {
	if user == "" {
		panic("user is empty")
	}
	return func(cfg *Config) {
		cfg.User = user
	}
}

func WithPassword(password string) Option {
	return func(cfg *Config) {
		cfg.Password = password
	}
}

func WithDatabase(database string) Option {
	if database == "" {
		panic("database is empty")
	}
	return func(cfg *Config) {
		cfg.Database = database
	}
}

func WithApplicationName(appName string) Option {
	return func(cfg *Config) {
		cfg.AppName = appName
	}
}

func WithConnParams(params map[string]interface{}) Option {
	return func(cfg *Config) {
		cfg.ConnParams = params
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.DialTimeout = timeout
		cfg.ReadTimeout = timeout
		cfg.WriteTimeout = timeout
	}
}

func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.DialTimeout = dialTimeout
	}
}

func WithReadTimeout(readTimeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.ReadTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.WriteTimeout = writeTimeout
	}
}

func WithDSN(dsn string) Option {
	return func(cfg *Config) {
		opts, err := parseDSN(dsn)
		if err != nil {
			panic(err)
		}
		for _, opt := range opts {
			opt(cfg)
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

func parseDSN(dsn string) ([]Option, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	q := queryOptions{q: u.Query()}
	var opts []Option

	switch u.Scheme {
	case "postgres", "postgresql":
		if u.Host != "" {
			addr := u.Host
			if !strings.Contains(addr, ":") {
				addr += ":5432"
			}
			opts = append(opts, WithAddr(addr))
		}

		if len(u.Path) > 1 {
			opts = append(opts, WithDatabase(u.Path[1:]))
		}

		if host := q.string("host"); host != "" {
			opts = append(opts, WithAddr(host))
			if host[0] == '/' {
				opts = append(opts, WithNetwork("unix"))
			}
		}
	case "unix":
		if len(u.Path) == 0 {
			return nil, fmt.Errorf("unix socket DSN requires a path: %s", dsn)
		}

		opts = append(opts, WithNetwork("unix"))
		if u.Host != "" {
			opts = append(opts, WithDatabase(u.Host))
		}
		opts = append(opts, WithAddr(u.Path))
	default:
		return nil, errors.New("pgdriver: invalid scheme: " + u.Scheme)
	}

	if u.User != nil {
		opts = append(opts, WithUser(u.User.Username()))
		if password, ok := u.User.Password(); ok {
			opts = append(opts, WithPassword(password))
		}
	}

	if appName := q.string("application_name"); appName != "" {
		opts = append(opts, WithApplicationName(appName))
	}

	switch sslMode := q.string("sslmode"); sslMode {
	case "verify-ca", "verify-full":
		opts = append(opts, WithTLSConfig(new(tls.Config)))
	case "allow", "prefer", "require", "":
		opts = append(opts, WithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	case "disable":
		opts = append(opts, WithInsecure(true))
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
		params := make(map[string]interface{}, len(rem))
		for k, v := range rem {
			params[k] = v
		}
		opts = append(opts, WithConnParams(params))
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

func (o *queryOptions) remaining() (map[string]string, error) {
	if o.err != nil {
		return nil, o.err
	}
	if len(o.q) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(o.q))
	for k, ss := range o.q {
		m[k] = ss[len(ss)-1]
	}
	return m, nil
}
