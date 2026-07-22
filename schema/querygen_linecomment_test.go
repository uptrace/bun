package schema

import (
	"strings"
	"testing"
)

// Regression test for the line-comment SQL injection class (CVE-2024-44906):
// a negative numeric argument that immediately follows a '-' must not produce
// a "--" sequence, which PostgreSQL would parse as the start of a line comment.
func TestQueryGen_Append_NegativeNumberDoesNotCreateLineComment(t *testing.T) {
	gen := nopQueryGen

	tests := []struct {
		name string
		expr string
		arg  any
	}{
		{"int", "1000-?", int(-500)},
		{"int32", "1000-?", int32(-500)},
		{"int64", "1000-?", int64(-500)},
		{"float32", "1000-?", float32(-1.5)},
		{"float64", "1000-?", float64(-1.5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := gen.FormatQuery(tt.expr, tt.arg)
			if strings.Contains(b, "--") {
				t.Fatalf("formatted query contains a line comment %q for arg %v (%T)", b, tt.arg, tt.arg)
			}
		})
	}
}
