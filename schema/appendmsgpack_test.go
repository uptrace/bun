package schema

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// xDialect emulates dialects such as SQLite and MySQL that expect binary
// literals in the X'...' form rather than PostgreSQL's '\x...' form.
type xDialect struct {
	*nopDialect
}

func (xDialect) AppendBytes(b, bs []byte) []byte {
	b = append(b, `X'`...)
	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)
	b = append(b, '\'')
	return b
}

// A msgpack field must be emitted using the binary literal syntax of the target
// dialect. Previously it was always encoded in PostgreSQL's '\x...' form, which
// other dialects store verbatim as text and then fail to decode on read back.
func TestAppendMsgpack_DialectSpecificLiteral(t *testing.T) {
	value := map[string]int{"something": 42}

	encoded, err := msgpack.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	rv := reflect.ValueOf(value)

	t.Run("postgres", func(t *testing.T) {
		gen := NewQueryGen(newNopDialect())
		got := string(appendMsgpack(gen, nil, rv))
		want := string(BaseDialect{}.AppendBytes(nil, encoded))
		if got != want {
			t.Fatalf("appendMsgpack = %q, want %q", got, want)
		}
		if !strings.HasPrefix(got, `'\x`) {
			t.Fatalf("appendMsgpack = %q, want a '\\x...' literal", got)
		}
	})

	t.Run("x-literal", func(t *testing.T) {
		d := xDialect{newNopDialect()}
		gen := NewQueryGen(d)
		got := string(appendMsgpack(gen, nil, rv))
		want := string(d.AppendBytes(nil, encoded))
		if got != want {
			t.Fatalf("appendMsgpack = %q, want %q", got, want)
		}
		if !strings.HasPrefix(got, "X'") {
			t.Fatalf("appendMsgpack = %q, want an X'...' literal", got)
		}
	})
}
