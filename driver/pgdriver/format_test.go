package pgdriver

import (
	"database/sql/driver"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatQuery(t *testing.T) {
	tests := []struct {
		query  string
		args   []any
		wanted string
	}{
		{
			query:  "select $1, $1, $2",
			args:   []any{"hello", int64(123)},
			wanted: "select 'hello', 'hello', 123",
		},
		{
			query:  "select '$1', $1",
			args:   []any{"hello"},
			wanted: "select '$1', 'hello'",
		},
		{
			query:  "select $1, $2",
			args:   []any{time.Unix(0, 0), math.NaN()},
			wanted: "select '1970-01-01 00:00:00+00:00', 'NaN'",
		},
		{
			query:  "select $1,$2,$3,$4",
			args:   []any{nil, "", []byte(nil), time.Time{}},
			wanted: "select NULL,'',NULL,NULL",
		},
		{
			query:  "select 1-$1, 1.0-$2, 1.0-$3",
			args:   []any{int64(-1), float64(-1.5), math.Inf(-1)},
			wanted: "select 1- -1, 1.0- -1.5, 1.0-'-Infinity'",
		},
		{
			query:  "select 1+$1, 1.0+$2",
			args:   []any{int64(-1), float64(-1.5)},
			wanted: "select 1+-1, 1.0+-1.5",
		},
		{
			query: "select 1-$1, $2",
			args:  []any{int64(-1), "foo\n;\nSELECT * FROM passwords;--"},
			// Without a space before the negative number, the first line ends in a comment
			wanted: `select 1- -1, 'foo
;
SELECT * FROM passwords;--'`,
		},
		{
			query:  "$1",
			args:   []any{int64(-1)},
			wanted: "-1",
		},
	}

	for _, test := range tests {
		query, err := formatQuery(test.query, namedValues(test.args...))
		require.NoError(t, err)
		require.Equal(t, test.wanted, query)
	}
}

func namedValues(args ...any) []driver.NamedValue {
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
