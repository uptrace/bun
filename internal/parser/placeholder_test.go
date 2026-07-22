package parser

import "testing"

func TestReadUntilPlaceholder(t *testing.T) {
	tests := []struct {
		query   string
		want    string // text before the first real placeholder
		wantOk  bool
		wantRem string // remaining after the consumed '?'
	}{
		{"a = ?", "a = ", true, ""},
		{"no placeholder", "no placeholder", false, ""},
		// '?' inside a single-quoted literal must be ignored.
		{"'why?' , ?", "'why?' , ", true, ""},
		{"note='huh?' AND id=?", "note='huh?' AND id=", true, ""},
		// doubled quote stays inside the literal.
		{"'it''s ?' , ?", "'it''s ?' , ", true, ""},
		// Double-quoted identifiers are intentionally NOT skipped: a '?' inside
		// them is a named placeholder (e.g. multi-tenant "?tenant".table).
		{`"?tenant".t = ?`, `"`, true, `tenant".t = ?`},
		// line comment.
		{"1 -- really?\n, ?", "1 -- really?\n, ", true, ""},
		// block comment (and nested).
		{"1 /* huh? */, ?", "1 /* huh? */, ", true, ""},
		{"1 /* a /* ? */ ? */, ?", "1 /* a /* ? */ ? */, ", true, ""},
		// real placeholder before any literal is still found first.
		{"? AND 'x?'", "", true, " AND 'x?'"},
	}

	for _, tt := range tests {
		p := NewString(tt.query)
		got, ok := p.ReadUntilPlaceholder()
		if string(got) != tt.want || ok != tt.wantOk {
			t.Errorf("ReadUntilPlaceholder(%q) = (%q, %v), want (%q, %v)",
				tt.query, got, ok, tt.want, tt.wantOk)
		}
		if rem := string(p.Remaining()); rem != tt.wantRem {
			t.Errorf("ReadUntilPlaceholder(%q) remaining = %q, want %q", tt.query, rem, tt.wantRem)
		}
	}
}
