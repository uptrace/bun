package schema

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// blobDialect stands in for any dialect whose byte literals are not spelled the
// PostgreSQL way. SQLite and MySQL use X'..' and MSSQL uses 0x.., so a msgpack
// column must be emitted through Dialect.AppendBytes rather than with a
// hard-coded syntax. It also mirrors the real dialects' nil => NULL rule, so a
// value that must never be NULL cannot pass here by accident.
type blobDialect struct {
	*nopDialect
}

func (blobDialect) AppendBytes(b, bs []byte) []byte {
	if bs == nil {
		return append(b, "NULL"...)
	}

	b = append(b, `X'`...)
	const digits = "0123456789abcdef"
	for _, c := range bs {
		b = append(b, digits[c>>4], digits[c&0x0f])
	}
	return append(b, '\'')
}

type item struct {
	Something int `msgpack:"something"`
}

// emptyMsgpack marshals to no bytes. msgpack still hands the result to the
// writer, so this is an empty value rather than a missing one.
type emptyMsgpack struct{}

func (emptyMsgpack) MarshalMsgpack() ([]byte, error) { return nil, nil }

// unwrittenMsgpack encodes successfully without ever invoking the writer, which
// the previous implementation spelled as SQL NULL.
type unwrittenMsgpack struct{}

func (unwrittenMsgpack) EncodeMsgpack(*msgpack.Encoder) error { return nil }

// failingMsgpack writes before it fails, so the staging writer is left dirty
// when the encode returns an error.
type failingMsgpack struct{}

func (failingMsgpack) EncodeMsgpack(enc *msgpack.Encoder) error {
	if err := enc.EncodeString("partial output"); err != nil {
		return err
	}
	return errors.New("encode failed")
}

func TestAppendMsgpack(t *testing.T) {
	value := item{Something: 1}

	encoded, err := msgpack.Marshal(value)
	if err != nil {
		t.Fatalf("msgpack.Marshal: %v", err)
	}

	t.Run("uses the dialect's byte literal syntax", func(t *testing.T) {
		gen := NewQueryGen(blobDialect{newNopDialect()})

		got := string(appendMsgpack(gen, nil, reflect.ValueOf(value)))

		want := string(blobDialect{}.AppendBytes(nil, encoded))
		if got != want {
			t.Errorf("appendMsgpack = %q, want %q", got, want)
		}
	})

	t.Run("appends to the existing query", func(t *testing.T) {
		gen := NewQueryGen(blobDialect{newNopDialect()})

		got := string(appendMsgpack(gen, []byte("VALUES ("), reflect.ValueOf(value)))

		want := "VALUES (" + string(blobDialect{}.AppendBytes(nil, encoded))
		if got != want {
			t.Errorf("appendMsgpack = %q, want %q", got, want)
		}
	})

	// PostgreSQL is the one dialect the previous hard-coded form happened to
	// match, so it is the one that must not move.
	t.Run("still emits PostgreSQL syntax for the base dialect", func(t *testing.T) {
		gen := NewQueryGen(newNopDialect())

		got := string(appendMsgpack(gen, nil, reflect.ValueOf(value)))

		want := string(BaseDialect{}.AppendBytes(nil, encoded))
		if got != want {
			t.Errorf("appendMsgpack = %q, want %q", got, want)
		}
		if got[:3] != `'\x` {
			t.Errorf("appendMsgpack = %q, want a '\\x.. literal", got)
		}
	})

	// An encoder that never writes and one that writes nothing are different
	// outcomes: the first has no value, the second has an empty one. SQL NULL and
	// an empty blob are not interchangeable for constraints, comparisons, or
	// scanning, so the staging writer has to record whether it was written to at
	// all rather than inferring it from an empty buffer.
	t.Run("distinguishes an unwritten encoder from a zero-length write", func(t *testing.T) {
		gen := NewQueryGen(newNopDialect())

		if got := string(appendMsgpack(gen, nil, reflect.ValueOf(unwrittenMsgpack{}))); got != "NULL" {
			t.Errorf("encoder that never wrote = %q, want NULL", got)
		}
		if got := string(appendMsgpack(gen, nil, reflect.ValueOf(emptyMsgpack{}))); got != `'\x'` {
			t.Errorf("marshaler that wrote no bytes = %q, want %q", got, `'\x'`)
		}
	})

	// A failing encoder writes before it fails, so its half-written output must
	// not reach the query. Comparing the whole result against the caller's prefix
	// plus the marker covers that in one assertion: any staged bytes would appear
	// between them, hex-encoded and therefore invisible to a substring check.
	// What happens to the writer afterwards is covered by TestMsgpackWriterReset,
	// which does not depend on pool reuse.
	t.Run("a failed encode yields only an error marker", func(t *testing.T) {
		gen := NewQueryGen(newNopDialect())

		got := string(appendMsgpack(gen, []byte("VALUES ("), reflect.ValueOf(failingMsgpack{})))

		if want := "VALUES (?!(encode failed)"; got != want {
			t.Errorf("appendMsgpack = %q, want %q", got, want)
		}
	})

	// Zero-length outcomes have to be spelled by the dialect too, or a branch
	// that special-cased them could hard-code PostgreSQL output and still pass
	// every test above.
	t.Run("routes zero-length outcomes through the dialect", func(t *testing.T) {
		gen := NewQueryGen(blobDialect{newNopDialect()})

		if got := string(appendMsgpack(gen, nil, reflect.ValueOf(emptyMsgpack{}))); got != "X''" {
			t.Errorf("marshaler that wrote no bytes = %q, want %q", got, "X''")
		}
		if got := string(appendMsgpack(gen, nil, reflect.ValueOf(unwrittenMsgpack{}))); got != "NULL" {
			t.Errorf("encoder that never wrote = %q, want NULL", got)
		}
	})

	// The whole path only matters if the struct tag actually routes here.
	t.Run("is selected by the msgpack struct tag", func(t *testing.T) {
		type tagged struct {
			Encoded item `bun:",msgpack"`
		}

		d := blobDialect{newNopDialect()}
		table := NewTables(d).Get(reflect.TypeOf(tagged{}))

		var field *Field
		for _, f := range table.Fields {
			if f.GoName == "Encoded" {
				field = f
			}
		}
		if field == nil {
			t.Fatal("Encoded field not found on the table")
		}

		model := tagged{Encoded: value}
		got := string(field.AppendValue(NewQueryGen(d), nil, reflect.ValueOf(model)))

		want := string(blobDialect{}.AppendBytes(nil, encoded))
		if got != want {
			t.Errorf("AppendValue = %q, want %q", got, want)
		}
	})
}

// TestMsgpackWriterReset checks the writer directly rather than through the
// pool: sync.Pool may hand back a different object, so a test that encodes twice
// and inspects the second result cannot prove anything about reuse.
func TestMsgpackWriterReset(t *testing.T) {
	var w msgpackWriter

	if _, err := w.Write([]byte("stale")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !w.written {
		t.Fatal("Write did not record that the writer was used")
	}

	w.reset()

	if w.written {
		t.Error("reset left the written flag set")
	}
	if w.buf.Len() != 0 {
		t.Errorf("reset left %d bytes behind", w.buf.Len())
	}
}

func BenchmarkAppendMsgpack(b *testing.B) {
	type payload struct {
		ID    int
		Name  string
		Tags  []string
		Bytes []byte
	}
	value := payload{
		ID:    1234567890,
		Name:  "representative-msgpack-tagged-column",
		Tags:  []string{"alpha", "beta", "gamma"},
		Bytes: make([]byte, 512),
	}
	rv := reflect.ValueOf(value)
	gen := NewQueryGen(newNopDialect())

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = appendMsgpack(gen, make([]byte, 0, 4096), rv)
	}
}
