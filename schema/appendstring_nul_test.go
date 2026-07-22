package schema

import (
	"strings"
	"testing"
)

func TestBaseDialectAppendString_NulFailsClosed(t *testing.T) {
	// A NUL byte must NOT be silently stripped; it must produce a formatting
	// error marker so the query fails instead of storing a mutated value.
	got := string(BaseDialect{}.AppendString(nil, "admin\x00x"))
	if strings.Contains(got, "adminx") {
		t.Fatalf("NUL was silently stripped: %q", got)
	}
	if !strings.Contains(got, "?!(") {
		t.Fatalf("AppendString with NUL = %q, want a formatting error marker", got)
	}
}

func TestBaseDialectAppendString_NoNul(t *testing.T) {
	tests := map[string]string{
		"abc":   `'abc'`,
		"it's":  `'it''s'`,
		"a\\b":  `'a\b'`, // backslash left as-is (standard_conforming_strings=on)
		"héllo": `'héllo'`,
	}
	for in, want := range tests {
		if got := string(BaseDialect{}.AppendString(nil, in)); got != want {
			t.Errorf("AppendString(%q) = %q, want %q", in, got, want)
		}
	}
}
