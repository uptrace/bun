package schema

import "testing"

func TestBaseDialectAppendString_NulFailsClosed(t *testing.T) {
	// A NUL byte must NOT be silently stripped; it must produce a formatting
	// error marker outside a quoted string so the query cannot remain valid.
	const want = "?!(bun: string contains a NUL byte (0x00))"
	for _, in := range []string{"\x00", "admin\x00x", "admin\x00'x"} {
		if got := string(BaseDialect{}.AppendString(nil, in)); got != want {
			t.Errorf("AppendString(%q) = %q, want %q", in, got, want)
		}
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
