package schema

import (
	"testing"

	"github.com/uptrace/bun/dialect"
)

// lcTestDialect is a non-nop dialect (so FormatQuery actually substitutes args)
// that reuses nopDialect's BaseDialect-backed Append* implementations.
type lcTestDialect struct {
	*nopDialect
}

func (lcTestDialect) Name() dialect.Name { return dialect.PG }

// Regression test for the line-comment SQL injection class (CVE-2024-44906):
// a negative numeric argument that immediately follows a '-' must not produce a
// "--" sequence, which PostgreSQL would parse as the start of a line comment. A
// space is inserted so e.g. "1000-?" with -500 renders as "1000- -500".
func TestQueryGen_Append_NegativeNumberDoesNotCreateLineComment(t *testing.T) {
	gen := NewQueryGen(lcTestDialect{newNopDialect()})

	tests := []struct {
		name string
		expr string
		arg  any
		want string
	}{
		{"int", "1000-?", int(-500), "1000- -500"},
		{"int32", "1000-?", int32(-500), "1000- -500"},
		{"int64", "1000-?", int64(-500), "1000- -500"},
		{"float32", "1000-?", float32(-1.5), "1000- -1.5"},
		{"float64", "1000-?", float64(-1.5), "1000- -1.5"},
		// non-adjacent minus is unaffected (no spurious space).
		{"spaced", "1000 - ?", int(-500), "1000 - -500"},
		// positive values are unaffected.
		{"positive", "1000-?", int(500), "1000-500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.FormatQuery(tt.expr, tt.arg)
			if got != tt.want {
				t.Fatalf("FormatQuery(%q, %v[%T]) = %q, want %q", tt.expr, tt.arg, tt.arg, got, tt.want)
			}
		})
	}
}
