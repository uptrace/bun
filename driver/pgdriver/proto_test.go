package pgdriver

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteStartupIncludesRequiredConnParams(t *testing.T) {
	netConn := new(bufferConn)
	cn := &Conn{
		conf: &Config{
			User:     "user",
			Database: "db",
			AppName:  "app",
			ConnParams: map[string]any{
				"client_encoding":              "utf-8",
				"standard_conforming_strings":  true,
				"search_path":                  "public",
				"statement_timeout":            1000,
				"ignored_param_with_nil_value": nil,
			},
		},
		netConn: netConn,
	}

	err := writeStartup(context.Background(), cn)
	require.NoError(t, err)

	b := netConn.Bytes()
	require.GreaterOrEqual(t, len(b), 8)
	require.Equal(t, len(b), int(binary.BigEndian.Uint32(b[:4])))
	require.Equal(t, int32(196608), int32(binary.BigEndian.Uint32(b[4:8])))

	params := readStartupParams(t, b[8:])
	require.Equal(t, "user", params["user"])
	require.Equal(t, "db", params["database"])
	require.Equal(t, "app", params["application_name"])
	require.Equal(t, "UTF8", params["client_encoding"])
	require.Equal(t, "on", params["standard_conforming_strings"])
	require.NotContains(t, params, "search_path")
	require.NotContains(t, params, "statement_timeout")
	require.NotContains(t, params, "ignored_param_with_nil_value")
}

func TestCheckRequiredParameter(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		value   string
		wantErr bool
	}{
		{
			name:  "standard_conforming_strings on",
			param: "standard_conforming_strings",
			value: "on",
		},
		{
			name:  "standard_conforming_strings true",
			param: "standard_conforming_strings",
			value: "true",
		},
		{
			name:    "standard_conforming_strings off",
			param:   "standard_conforming_strings",
			value:   "off",
			wantErr: true,
		},
		{
			name:  "client_encoding UTF8",
			param: "client_encoding",
			value: "UTF8",
		},
		{
			name:  "client_encoding utf-8",
			param: "client_encoding",
			value: "utf-8",
		},
		{
			name:  "client_encoding unicode alias",
			param: "client_encoding",
			value: "Unicode",
		},
		{
			name:    "client_encoding latin1",
			param:   "client_encoding",
			value:   "LATIN1",
			wantErr: true,
		},
		{
			name:  "unrelated parameter",
			param: "server_version",
			value: "16.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkRequiredParameter(test.param, test.value)
			if test.wantErr {
				require.ErrorIs(t, err, errRequiresParameter)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckParameterStatus(t *testing.T) {
	err := checkParameterStatus([]byte("client_encoding\x00LATIN1\x00"))
	require.ErrorIs(t, err, errRequiresParameter)

	err = checkParameterStatus([]byte("client_encoding\x00UTF8\x00"))
	require.NoError(t, err)
}

func readStartupParams(t *testing.T, b []byte) map[string]string {
	t.Helper()

	params := make(map[string]string)
	for {
		key := readCString(t, &b)
		if key == "" {
			require.Empty(t, b)
			return params
		}

		value := readCString(t, &b)
		params[key] = value
	}
}

func readCString(t *testing.T, b *[]byte) string {
	t.Helper()

	i := bytes.IndexByte(*b, 0)
	require.NotEqual(t, -1, i)

	s := string((*b)[:i])
	*b = (*b)[i+1:]
	return s
}

type bufferConn struct {
	bytes.Buffer
}

func (c *bufferConn) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (c *bufferConn) Close() error {
	return nil
}

func (c *bufferConn) LocalAddr() net.Addr {
	return testAddr("local")
}

func (c *bufferConn) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (c *bufferConn) SetDeadline(time.Time) error {
	return nil
}

func (c *bufferConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *bufferConn) SetWriteDeadline(time.Time) error {
	return nil
}

type testAddr string

func (a testAddr) Network() string {
	return string(a)
}

func (a testAddr) String() string {
	return string(a)
}
