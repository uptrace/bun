package pgdialect

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/schema"
)

// This file guards against behavior changes in the PostgreSQL array appender
// for types that are not the standard library uuid.UUID.

// valuerPtrString implements driver.Valuer on the pointer receiver only, so
// the value form must keep using the default (JSON) element appender.
type valuerPtrString struct{ s string }

func (v *valuerPtrString) Value() (driver.Value, error) {
	if v == nil {
		// A nil-aware valuer must still be consulted for a nil pointer
		// element rather than being replaced by SQL NULL.
		return "NIL-SENTINEL", nil
	}
	return "VAL", nil
}

// textMarshalingStruct has a text encoder and is not a UUID, so an array of it
// keeps the default JSON element appender.
type textMarshalingStruct struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (v textMarshalingStruct) MarshalText() ([]byte, error) { return []byte("text"), nil }

func (v textMarshalingStruct) MarshalJSON() ([]byte, error) {
	return []byte(`[1,"x"]`), nil
}

func TestArrayCompatValuerDispatch(t *testing.T) {
	gen := schema.NewQueryGen(New())

	tcases := []struct {
		input any
		out   string
	}{
		// Value elements must NOT pick up the pointer-receiver Value method.
		{[]valuerPtrString{{"raw"}}, `'{'VAL'}'`},
		{[]*valuerPtrString{{"raw"}}, `'{"VAL"}'`},
		// A nil element must consult the nil-aware valuer.
		{[]*valuerPtrString{nil}, `'{"NIL-SENTINEL"}'`},
	}

	for _, tcase := range tcases {
		out, err := Array(tcase.input).AppendQuery(gen, nil)
		require.NoError(t, err, "input %T", tcase.input)
		require.Equal(t, tcase.out, string(out), "input %T", tcase.input)
	}
}

func TestArrayCompatNonUUIDTextTypes(t *testing.T) {
	gen := schema.NewQueryGen(New())

	out, err := Array([]textMarshalingStruct{{A: 1, B: "x"}}).AppendQuery(gen, nil)
	require.NoError(t, err)
	require.Equal(t, `'{'[1,"x"]'}'`, string(out),
		"a text-marshaling struct element keeps the JSON element appender")
}

func TestArrayCompatByteArrays(t *testing.T) {
	gen := schema.NewQueryGen(New())

	out, err := Array([][16]byte{{1, 2, 3}}).AppendQuery(gen, nil)
	require.NoError(t, err)
	require.Equal(t, `'{"\\x01020300000000000000000000000000"}'`, string(out))
}
