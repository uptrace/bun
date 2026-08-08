package schema

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"testing"
)

// badValuer reproduces the driver.Valuer error-message SQL injection (bun
// issue #1307): Value() returns an error whose message is crafted to close
// the "?!(...)" marker and append arbitrary SQL.
type badValuer struct{}

func (badValuer) Value() (driver.Value, error) {
	return nil, errors.New(`pwned'); INSERT INTO pwned VALUES (999); --`)
}

func TestAppendDriverValue_ErrorIsEscaped(t *testing.T) {
	gen := NewQueryGen(newNopDialect())
	got := string(appendDriverValue(gen, nil, reflect.ValueOf(badValuer{})))

	// The embedded quote must be doubled, keeping the injected text inside
	// the string literal instead of letting it break out.
	want := `?!('pwned''); INSERT INTO pwned VALUES (999); --')`
	if got != want {
		t.Fatalf("appendDriverValue(badValuer) = %q, want %q", got, want)
	}
}

// A driver.Valuer that succeeds must be unaffected by the error-escaping path.
func TestAppendDriverValue_NoError(t *testing.T) {
	gen := NewQueryGen(newNopDialect())
	got := string(appendDriverValue(gen, nil, reflect.ValueOf(sql.NullString{String: "ok", Valid: true})))

	want := `'ok'`
	if got != want {
		t.Fatalf("appendDriverValue(NullString) = %q, want %q", got, want)
	}
}
