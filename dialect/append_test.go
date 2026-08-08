package dialect

import (
	"errors"
	"strings"
	"testing"
)

// fakeEscaper stands in for schema.Dialect.AppendString (which dialect
// cannot import without creating an import cycle): quote the string and
// double embedded single quotes, the same as the real SQL dialects.
type fakeEscaper struct{}

func (fakeEscaper) AppendString(b []byte, s string) []byte {
	b = append(b, '\'')
	for _, r := range s {
		if r == '\'' {
			b = append(b, '\'', '\'')
			continue
		}
		b = append(b, string(r)...)
	}
	b = append(b, '\'')
	return b
}

// Regression test for the driver.Valuer error-message SQL injection (bun
// issue #1307): err.Error() used to be appended to the "?!(...)" marker raw,
// so a crafted message could close the marker and inject arbitrary SQL.
func TestAppendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"plain", errors.New("boom"), `?!('boom')`},
		// Closing the marker with ")" no longer works once the message is a
		// quoted string literal.
		{"paren breakout attempt", errors.New(`1)); DROP TABLE t; --`), `?!('1)); DROP TABLE t; --')`},
		// A quote in the message must be doubled, not left to close the
		// string literal early.
		{"embedded quote", errors.New(`it's bad`), `?!('it''s bad')`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(AppendError(nil, fakeEscaper{}, tt.err))
			if got != tt.want {
				t.Fatalf("AppendError(%v) = %q, want %q", tt.err, got, tt.want)
			}
			if strings.Count(got, "?!(") != 1 {
				t.Fatalf("AppendError(%v) = %q, want exactly one formatting error marker", tt.err, got)
			}
		})
	}
}
