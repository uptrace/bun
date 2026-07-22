package schema

import (
	"encoding/json"
	"testing"
)

func TestBaseDialectAppendJSON(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// A backslash immediately followed by a single quote must NOT produce an
		// un-doubled quote (which would break out of the SQL string literal).
		{`{"x":"a\'; DROP TABLE t; --"}`, `'{"x":"a\''; DROP TABLE t; --"}'`},
		// Valid encoding/json escapes are preserved.
		{`{"k":"a\"b\\c"}`, `'{"k":"a\"b\\c"}'`},
		// A raw single quote is doubled for SQL.
		{`{"k":"it's"}`, `'{"k":"it''s"}'`},
	}

	for _, tt := range tests {
		got := string(BaseDialect{}.AppendJSON(nil, []byte(tt.in)))
		if got != tt.want {
			t.Errorf("AppendJSON(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// The default ORM path pre-marshals with encoding/json, which never emits the
// byte pair `\'`; such output must embed verbatim (only wrapped in quotes here,
// since these values contain no single quote).
func TestBaseDialectAppendJSON_EncodingJSONRoundTrips(t *testing.T) {
	for _, v := range []any{
		map[string]string{"msg": `a"b\c/d`},
		map[string]string{"path": `C:\tmp\x`},
		[]string{"line1\nline2", "tab\tend"},
	} {
		jsonb, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		got := string(BaseDialect{}.AppendJSON(nil, jsonb))
		want := "'" + string(jsonb) + "'"
		if got != want {
			t.Errorf("AppendJSON(%s) = %s, want %s", jsonb, got, want)
		}
	}
}
