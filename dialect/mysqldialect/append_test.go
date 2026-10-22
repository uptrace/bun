package mysqldialect

import (
	"testing"

	"github.com/uptrace/bun/schema"
)

// Regression test for #1443: a NUL byte used to be stripped, so a different
// value was stored than the one supplied.
func TestAppendStringNul(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"a\x00b", `'a\0b'`},
		{"\x00", `'\0'`},
		{"\x000", `'\00'`}, // NUL followed by the digit 0
		{"a\\\x00", `'a\\\0'`},
		{"it's", `'it''s'`},
	}

	d := New()
	for _, test := range tests {
		if got := string(d.AppendString(nil, test.in)); got != test.want {
			t.Errorf("AppendString(%q) = %s, want %s", test.in, got, test.want)
		}
	}
}

// MySQL stores NUL, so an executed query must not refuse it.
func TestCheckStringAcceptsNul(t *testing.T) {
	gen := schema.NewQueryGen(New()).WithErrorSink()

	const want = `name = 'a\0b'`
	if got := gen.FormatQuery("name = ?", "a\x00b"); got != want {
		t.Errorf("FormatQuery = %s, want %s", got, want)
	}
	if err := gen.Err(); err != nil {
		t.Errorf("Err = %v, want nil", err)
	}
}
