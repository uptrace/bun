package pgdriver

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatQuery(t *testing.T) {
	tests := []struct {
		query  string
		args   []interface{}
		wanted string
	}{
		{
			query:  "select $1, $1, $2",
			args:   []interface{}{"hello", int64(123)},
			wanted: "select 'hello', 'hello', 123",
		},
		{
			query:  "select '$1', $1",
			args:   []interface{}{"hello"},
			wanted: "select '$1', 'hello'",
		},
	}

	for _, test := range tests {
		query, err := formatQuery(test.query, namedValues(test.args...))
		require.NoError(t, err)
		require.Equal(t, test.wanted, query)
	}
}

func namedValues(args ...interface{}) []driver.NamedValue {
	vals := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		vals[i] = driver.NamedValue{Value: arg}
	}
	return vals
}

func BenchmarkFormatQuery(b *testing.B) {
	query := "select $1, $1, $2"
	args := namedValues("hello", 123.456)

	for i := 0; i < b.N; i++ {
		_, err := formatQuery(query, args)
		if err != nil {
			b.Fatal(err)
		}
	}
}
