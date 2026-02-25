package pgdriver_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
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
				Network:       "tcp",
				Addr:          "localhost:5432",
				User:          "user",
				Password:      "password",
				Database:      "testDatabase",
				DialTimeout:   5 * time.Second,
				ReadTimeout:   10 * time.Second,
				WriteTimeout:  5 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
			},
		},
		{
			dsn: "postgres://user:password@localhost:5432/testDatabase?sslmode=disable&dial_timeout=1&read_timeout=2s&write_timeout=3",
			cfg: &pgdriver.Config{
				Network:       "tcp",
				Addr:          "localhost:5432",
				User:          "user",
				Password:      "password",
				Database:      "testDatabase",
				DialTimeout:   1 * time.Second,
				ReadTimeout:   2 * time.Second,
				WriteTimeout:  3 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
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
				ConnParams: map[string]any{
					"search_path": "foo",
				},
				DialTimeout:   5 * time.Second,
				ReadTimeout:   10 * time.Second,
				WriteTimeout:  5 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
			},
		},
		{
			dsn: "postgres://user:password@app.xxx.us-east-1.rds.amazonaws.com:5432/test?sslmode=disable",
			cfg: &pgdriver.Config{
				Network:       "tcp",
				Addr:          "app.xxx.us-east-1.rds.amazonaws.com:5432",
				User:          "user",
				Password:      "password",
				Database:      "test",
				DialTimeout:   5 * time.Second,
				ReadTimeout:   10 * time.Second,
				WriteTimeout:  5 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
			},
		},
		{
			dsn: "postgres://user:password@/dbname?host=/var/run/postgresql/.s.PGSQL.5432",
			cfg: &pgdriver.Config{
				Network:       "unix",
				Addr:          "/var/run/postgresql/.s.PGSQL.5432",
				User:          "user",
				Password:      "password",
				Database:      "dbname",
				DialTimeout:   5 * time.Second,
				ReadTimeout:   10 * time.Second,
				WriteTimeout:  5 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
			},
		},
		{
			dsn: "unix://user:pass@dbname/var/run/postgresql/.s.PGSQL.5432",
			cfg: &pgdriver.Config{
				Network:       "unix",
				Addr:          "/var/run/postgresql/.s.PGSQL.5432",
				User:          "user",
				Password:      "pass",
				Database:      "dbname",
				DialTimeout:   5 * time.Second,
				ReadTimeout:   10 * time.Second,
				WriteTimeout:  5 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
			},
		},
		{
			dsn: "postgres://user:password@localhost:5432/testDatabase?connect_timeout=3",
			cfg: &pgdriver.Config{
				Network:       "tcp",
				Addr:          "localhost:5432",
				User:          "user",
				Password:      "password",
				Database:      "testDatabase",
				DialTimeout:   3 * time.Second,
				ReadTimeout:   10 * time.Second,
				WriteTimeout:  5 * time.Second,
				EnableTracing: true,
				BufferSize:    4096,
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

func TestParseSSLClientCertInDSN(t *testing.T) {
	cert, key := generateTestCertFiles(t)
	pair, err := tls.LoadX509KeyPair(cert, key)
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://user:password@localhost:5432/testDatabase?sslmode=require&sslcert=%s&sslkey=%s", cert, key)
	c := pgdriver.NewConnector(pgdriver.WithDSN(dsn))

	cfg := c.Config()
	require.NotNil(t, cfg.TLSConfig)
	require.Len(t, cfg.TLSConfig.Certificates, 1)
	require.Len(t, cfg.TLSConfig.Certificates[0].Certificate, 1)
	require.Equal(t, pair.Certificate[0], cfg.TLSConfig.Certificates[0].Certificate[0])
}

func generateTestCertFiles(t *testing.T) (certPath, keyPath string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir() // automatically cleaned up after test

	certPath = filepath.Join(dir, "client.crt")
	keyPath = filepath.Join(dir, "client.key")

	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}

	return certPath, keyPath
}
