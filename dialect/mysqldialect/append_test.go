package mysqldialect

import (
	"testing"

	"github.com/uptrace/bun/schema"
)

// mysqldialect accepts every string, so an executed query must not refuse it.
func TestCheckStringAcceptsNul(t *testing.T) {
	gen := schema.NewQueryGen(New()).WithErrorSink()

	const want = `name = 'ab'`
	if got := gen.FormatQuery("name = ?", "a\x00b"); got != want {
		t.Errorf("FormatQuery = %s, want %s", got, want)
	}
	if err := gen.Err(); err != nil {
		t.Errorf("Err = %v, want nil", err)
	}
}
