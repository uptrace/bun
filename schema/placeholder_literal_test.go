package schema

import (
	"testing"

	"github.com/uptrace/bun/dialect"
)

// testDialect is a non-nop dialect (so FormatQuery actually substitutes args)
// that reuses nopDialect's BaseDialect-backed Append* implementations.
type testDialect struct {
	*nopDialect
}

func (testDialect) Name() dialect.Name { return dialect.PG }

func newTestQueryGen() QueryGen {
	return NewQueryGen(testDialect{newNopDialect()})
}

// Regression test: a '?' inside a string literal or comment in the query template
// must not be treated as a bind placeholder (which used to shift every subsequent
// positional argument).
func TestFormatQuery_PlaceholderInsideLiteralOrComment(t *testing.T) {
	gen := newTestQueryGen()

	tests := []struct {
		query string
		args  []any
		want  string
	}{
		// '?' inside a literal is left intact; the real placeholders bind correctly.
		{
			`note='huh?' AND org_id=? AND secret=?`,
			[]any{42, 99},
			`note='huh?' AND org_id=42 AND secret=99`,
		},
		{`'why?' , ?`, []any{7}, `'why?' , 7`},
		// doubled quote inside the literal.
		{`'it''s ?' , ?`, []any{5}, `'it''s ?' , 5`},
		// line and block comments.
		{"x=? -- really?\n", []any{1}, "x=1 -- really?\n"},
		{`x=? /* huh? */`, []any{2}, `x=2 /* huh? */`},
		// no false positives: normal placeholders still work.
		{`a=? AND b=?`, []any{1, 2}, `a=1 AND b=2`},
	}

	for _, tt := range tests {
		got := gen.FormatQuery(tt.query, tt.args...)
		if got != tt.want {
			t.Errorf("FormatQuery(%q, %v) = %q, want %q", tt.query, tt.args, got, tt.want)
		}
	}
}

// A named placeholder inside a double-quoted identifier must still be
// substituted (bun's multi-tenant pattern: "?tenant".table via WithNamedArg).
// Double-quoted identifiers must NOT be treated like string literals.
func TestFormatQuery_NamedPlaceholderInQuotedIdentifier(t *testing.T) {
	gen := newTestQueryGen().WithNamedArg("tenant", Safe("public"))

	got := gen.FormatQuery(`"?tenant".recipes`)
	want := `"public".recipes`
	if got != want {
		t.Fatalf("FormatQuery(%q) = %q, want %q", `"?tenant".recipes`, got, want)
	}
}
