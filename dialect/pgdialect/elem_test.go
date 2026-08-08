package pgdialect

import (
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/uptrace/bun/schema"
)

// badValuer reproduces the driver.Valuer error-message SQL injection (bun
// issue #1307) for array/range elements, whose escaping goes through
// appendElem instead of schema's appendDriverValue.
type badValuer struct{}

func (badValuer) Value() (driver.Value, error) {
	return nil, errors.New(`x'); DROP TABLE t; --`)
}

func TestAppendElem_ValuerErrorIsEscaped(t *testing.T) {
	got := string(appendElem(nil, pgDialect, badValuer{}))

	if !strings.Contains(got, "?!(") {
		t.Fatalf("appendElem(badValuer) = %q, want a formatting error marker", got)
	}
	// The quote in the payload must be doubled, not left to close the
	// string literal (and therefore the marker) early.
	if strings.Contains(got, "x');") {
		t.Fatalf("appendElem(badValuer) = %q, quote was not escaped", got)
	}
	if !strings.Contains(got, "x'');") {
		t.Fatalf("appendElem(badValuer) = %q, want escaped payload", got)
	}
}

func TestAppendElem_UnsupportedTypeErrorIsEscaped(t *testing.T) {
	got := string(appendElem(nil, pgDialect, struct{ X string }{}))
	if !strings.Contains(got, "?!(") || !strings.Contains(got, "struct { X string }") {
		t.Fatalf("appendElem(unsupported) = %q", got)
	}
}

// A plain value is unaffected by the error-escaping path.
func TestAppendElem_NoError(t *testing.T) {
	got := string(appendElem(nil, pgDialect, int64(42)))
	if got != "42" {
		t.Fatalf("appendElem(int64) = %q, want %q", got, "42")
	}
}

// Range.AppendQuery must thread the real dialect through to appendElem so a
// driver.Valuer error in a range bound gets escaped the same way.
func TestRangeAppendQuery_ValuerErrorIsEscaped(t *testing.T) {
	r := NewRange[driver.Valuer](badValuer{}, badValuer{})
	gen := schema.NewQueryGen(pgDialect)

	got, err := r.AppendQuery(gen, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "x');") {
		t.Fatalf("Range.AppendQuery(badValuer) = %q, quote was not escaped", got)
	}
	if !strings.Contains(string(got), "x'');") {
		t.Fatalf("Range.AppendQuery(badValuer) = %q, want escaped payload", got)
	}
}
