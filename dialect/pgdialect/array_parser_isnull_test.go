package pgdialect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArrayParserIsNull pins the parser's ability to tell an unquoted NULL from
// a quoted string, including an empty one. ReadSubstring reuses its buffer, so
// an empty quoted element is nil or non-nil depending on what preceded it, and
// IsNull must be correct in both positions.
func TestArrayParserIsNull(t *testing.T) {
	tcases := []struct {
		src  string
		want []struct {
			elem   string
			isNull bool
		}
	}{
		{
			src: `{NULL}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"", true}},
		},
		{
			src: `{""}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"", false}},
		},
		{
			src: `{a}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"a", false}},
		},
		{
			src: `{"a"}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"a", false}},
		},
		{
			src: `{NULL,""}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"", true}, {"", false}},
		},
		{
			src: `{"",NULL}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"", false}, {"", true}},
		},
		{
			src: `{"a",NULL}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"a", false}, {"", true}},
		},
		{
			src: `{NULL,"a"}`,
			want: []struct {
				elem   string
				isNull bool
			}{{"", true}, {"a", false}},
		},
	}

	for _, tcase := range tcases {
		t.Run(tcase.src, func(t *testing.T) {
			p := newArrayParser([]byte(tcase.src))
			for _, want := range tcase.want {
				require.True(t, p.Next(), "src %s", tcase.src)
				require.Equal(t, want.isNull, p.IsNull(), "IsNull for %s", tcase.src)
				require.Equal(t, want.elem, string(p.Elem()), "Elem for %s", tcase.src)
			}
			require.False(t, p.Next(), "src %s", tcase.src)
			require.NoError(t, p.Err(), "src %s", tcase.src)
		})
	}
}

// TestArrayParserEmptyQuotedStringNotNil pins the part of the behavior that
// IsNull is responsible for: an empty quoted string is never reported as NULL,
// in any position, even though its element may be nil or an empty slice
// depending on whether the reusable substring buffer had grown. The element
// itself is deliberately left alone, so other consumers keep their existing
// behavior.
func TestArrayParserEmptyQuotedStringNotNil(t *testing.T) {
	tcases := []struct {
		src  string
		want []string
	}{
		{`{""}`, []string{""}},
		{`{"a",""}`, []string{"a", ""}},
		{`{"",""}`, []string{"", ""}},
		{`{"",a}`, []string{"", "a"}},
		{`{a,""}`, []string{"a", ""}},
	}

	for _, tcase := range tcases {
		t.Run(tcase.src, func(t *testing.T) {
			p := newArrayParser([]byte(tcase.src))
			var got []string
			for p.Next() {
				require.False(t, p.IsNull(), "src %s: an empty quoted string is not NULL", tcase.src)
				got = append(got, string(p.Elem()))
			}
			require.NoError(t, p.Err(), "src %s", tcase.src)
			require.Equal(t, tcase.want, got, "src %s", tcase.src)
		})
	}
}
