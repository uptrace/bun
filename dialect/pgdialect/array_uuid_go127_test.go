//go:build go1.27

package pgdialect

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/schema"
)

var (
	testUUIDVal   = uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	testUUIDVal2  = uuid.MustParse("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")
	testUUIDText  = `"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"`
	testUUIDText2 = `"b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"`
)

func TestUUIDArrayAppend(t *testing.T) {
	gen := schema.NewQueryGen(New())

	tcases := []struct {
		input any
		out   string
	}{
		{
			input: []uuid.UUID{testUUIDVal},
			out:   `'{` + testUUIDText + `}'`,
		},
		{
			input: []uuid.UUID{testUUIDVal, testUUIDVal2},
			out:   `'{` + testUUIDText + `,` + testUUIDText2 + `}'`,
		},
		{
			// The nil UUID is a UUID value, not SQL NULL.
			input: []uuid.UUID{uuid.Nil()},
			out:   `'{"00000000-0000-0000-0000-000000000000"}'`,
		},
		{
			input: []*uuid.UUID{&testUUIDVal, nil},
			out:   `'{` + testUUIDText + `,NULL}'`,
		},
		{
			// A nil slice is SQL NULL, as for any other slice.
			input: []uuid.UUID(nil),
			out:   `NULL`,
		},
	}

	for _, tcase := range tcases {
		out, err := Array(tcase.input).AppendQuery(gen, []byte{})
		require.NoError(t, err)
		require.Equal(t, tcase.out, string(out), "input %T", tcase.input)
	}
}

func TestUUIDArrayScan(t *testing.T) {
	t.Run("value elements", func(t *testing.T) {
		tcases := []struct {
			src  string
			want []uuid.UUID
		}{
			{`{}`, []uuid.UUID{}},
			{`{` + testUUIDText + `}`, []uuid.UUID{testUUIDVal}},
			{`{` + testUUIDText + `,` + testUUIDText2 + `}`, []uuid.UUID{testUUIDVal, testUUIDVal2}},
			{`{NULL}`, []uuid.UUID{uuid.Nil()}},
			{`{NULL,` + testUUIDText + `}`, []uuid.UUID{uuid.Nil(), testUUIDVal}},
			{`{` + testUUIDText + `,NULL}`, []uuid.UUID{testUUIDVal, uuid.Nil()}},
			{`{NULL,NULL}`, []uuid.UUID{uuid.Nil(), uuid.Nil()}},
		}

		for _, tcase := range tcases {
			// Fresh destination.
			var dest []uuid.UUID
			require.NoError(t, Array(&dest).Scan(tcase.src), "src %s", tcase.src)
			require.Equal(t, tcase.want, dest, "fresh dest for %s", tcase.src)

			// Reused destination: stale elements must not survive.
			var reused []uuid.UUID
			require.NoError(t, Array(&reused).Scan(`{`+testUUIDText+`,`+testUUIDText2+`}`))
			require.NoError(t, Array(&reused).Scan(tcase.src), "src %s", tcase.src)
			require.Equal(t, tcase.want, reused, "reused dest for %s", tcase.src)
		}
	})

	t.Run("pointer elements", func(t *testing.T) {
		tcases := []struct {
			src  string
			want []*uuid.UUID
		}{
			{`{}`, []*uuid.UUID{}},
			{`{` + testUUIDText + `}`, []*uuid.UUID{&testUUIDVal}},
			{`{NULL}`, []*uuid.UUID{nil}},
			{`{NULL,` + testUUIDText + `}`, []*uuid.UUID{nil, &testUUIDVal}},
			{`{` + testUUIDText + `,NULL}`, []*uuid.UUID{&testUUIDVal, nil}},
		}

		for _, tcase := range tcases {
			var dest []*uuid.UUID
			require.NoError(t, Array(&dest).Scan(tcase.src), "src %s", tcase.src)
			require.Len(t, dest, len(tcase.want), "src %s", tcase.src)
			for i := range tcase.want {
				if tcase.want[i] == nil {
					require.Nil(t, dest[i], "element %d of %s must stay nil", i, tcase.src)
					continue
				}
				require.NotNil(t, dest[i], "element %d of %s", i, tcase.src)
				require.Equal(t, *tcase.want[i], *dest[i], "element %d of %s", i, tcase.src)
			}
		}
	})

	t.Run("whole array null", func(t *testing.T) {
		dest := []uuid.UUID{testUUIDVal}
		require.NoError(t, Array(&dest).Scan(nil))
		require.Nil(t, dest)

		ptrs := []*uuid.UUID{&testUUIDVal}
		require.NoError(t, Array(&ptrs).Scan(nil))
		require.Nil(t, ptrs)
	})

	t.Run("quoted empty string is rejected", func(t *testing.T) {
		// A quoted empty string is not NULL and not a valid UUID. ReadSubstring
		// reuses its buffer, so the element is nil or non-nil depending on the
		// preceding token; both positions must be rejected.
		for _, src := range []string{
			`{""}`,
			`{` + testUUIDText + `,""}`,
			`{"",` + testUUIDText + `}`,
		} {
			var dest []uuid.UUID
			require.Error(t, Array(&dest).Scan(src), "src %s", src)

			var ptrs []*uuid.UUID
			require.Error(t, Array(&ptrs).Scan(src), "src %s", src)
		}
	})

	t.Run("malformed 16-character text is rejected", func(t *testing.T) {
		// Array elements are text, so this must not be read as raw bytes.
		for _, src := range []string{
			`{"0123456789abcdef"}`,
			`{"abcdef0123456789"}`,
		} {
			var dest []uuid.UUID
			require.Error(t, Array(&dest).Scan(src), "src %s must not be raw bytes", src)

			var ptrs []*uuid.UUID
			require.Error(t, Array(&ptrs).Scan(src), "src %s must not be raw bytes", src)
		}
	})

	t.Run("fixed size array", func(t *testing.T) {
		var dest [1]uuid.UUID
		require.NoError(t, Array(&dest).Scan(`{`+testUUIDText+`}`))
		require.Equal(t, [1]uuid.UUID{testUUIDVal}, dest)

		require.NoError(t, Array(&dest).Scan(`{NULL}`))
		require.Equal(t, [1]uuid.UUID{uuid.Nil()}, dest)

		var ptrs [1]*uuid.UUID
		require.NoError(t, Array(&ptrs).Scan(`{`+testUUIDText+`}`))
		require.Equal(t, [1]*uuid.UUID{&testUUIDVal}, ptrs)

		require.NoError(t, Array(&ptrs).Scan(`{NULL}`))
		require.Equal(t, [1]*uuid.UUID{nil}, ptrs)

		// More elements than the array can hold is an error, not a silent
		// truncation. This holds for values and for NULL elements.
		for _, src := range []string{
			`{` + testUUIDText + `,` + testUUIDText2 + `}`,
			`{` + testUUIDText + `,NULL}`,
			`{NULL,` + testUUIDText + `}`,
			`{NULL,NULL}`,
		} {
			var overflowValues = [1]uuid.UUID{testUUIDVal}
			err := Array(&overflowValues).Scan(src)
			require.Error(t, err, "src %s must not be truncated", src)
			require.Contains(t, err.Error(), "more elements than")
			require.Equal(t, [1]uuid.UUID{testUUIDVal}, overflowValues,
				"src %s must leave the array untouched", src)

			var overflowPtrs = [1]*uuid.UUID{&testUUIDVal}
			err = Array(&overflowPtrs).Scan(src)
			require.Error(t, err, "src %s must not be truncated", src)
			require.Contains(t, err.Error(), "more elements than")
			require.Equal(t, [1]*uuid.UUID{&testUUIDVal}, overflowPtrs,
				"src %s must leave the array untouched", src)
		}

		// An empty array value writes nothing and leaves the array as it was.
		require.NoError(t, Array(&ptrs).Scan(`{`+testUUIDText+`}`))
		require.Equal(t, [1]*uuid.UUID{&testUUIDVal}, ptrs)

		require.NoError(t, Array(&ptrs).Scan(`{}`))
		require.Equal(t, [1]*uuid.UUID{&testUUIDVal}, ptrs,
			"an empty array value must write nothing")
	})

	t.Run("invalid element preserves nothing", func(t *testing.T) {
		var dest []uuid.UUID
		require.Error(t, Array(&dest).Scan(`{`+testUUIDText+`,"not-a-uuid"}`))
	})
}
