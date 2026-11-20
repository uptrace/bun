package pgdialect

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArrayParser(t *testing.T) {
	tests := []struct {
		s   string
		els []string
	}{
		{`{}`, []string{}},
		{`{""}`, []string{""}},
		{`{"\\"}`, []string{`\`}},
		{`{"''"}`, []string{"'"}},
		{`{{"'\"{}"}}`, []string{`{"'\"{}"}`}},
		{`{"'\"{}"}`, []string{`'"{}`}},

		{"{1,2}", []string{"1", "2"}},
		{"{1,NULL}", []string{"1", ""}},
		{`{"1","2"}`, []string{"1", "2"}},
		{`{"{1}","{2}"}`, []string{"{1}", "{2}"}},
		{`{[1,2),[3,4)}`, []string{"[1,2)", "[3,4)"}},

		{`[]`, []string{}},
		{`[{"'\"[]"}]`, []string{`{"'\"[]"}`}},
		{`[{"id": 1}, {"id":2, "name":"bob"}]`, []string{"{\"id\": 1}", "{\"id\":2, \"name\":\"bob\"}"}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			p := newArrayParser([]byte(test.s))

			got := make([]string, 0)
			for p.Next() {
				elem := p.Elem()
				got = append(got, string(elem))
			}

			require.NoError(t, p.Err())
			require.Equal(t, test.els, got)
		})
	}
}

func TestArrayParserInvalid(t *testing.T) {
	for _, s := range []string{"", "{", "}", "1"} {
		t.Run(fmt.Sprintf("%q", s), func(t *testing.T) {
			p := newArrayParser([]byte(s))
			require.False(t, p.Next())
			require.Error(t, p.Err())
		})
	}
}

func TestArrayScanTextElementsWithBrackets(t *testing.T) {
	// PostgreSQL only quotes a text element that is empty, NULL, or contains a
	// delimiter, brace, quote, backslash or white space, so these arrive unquoted.
	tests := []struct {
		s    string
		want []string
	}{
		{`{(555)123-4567}`, []string{"(555)123-4567"}},
		{`{[WIP]fix,done}`, []string{"[WIP]fix", "done"}},
		{`{(a,b)}`, []string{"(a", "b)"}},
		{`{[1,2),[3,4)}`, []string{"[1", "2)", "[3", "4)"}},
		{`{],x}`, []string{"]", "x"}},
		{`{x,]}`, []string{"x", "]"}},
		{`{"(a b",c)}`, []string{"(a b", "c)"}},
	}

	type label string

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			var got []string
			require.NoError(t, Array(&got).Scan([]byte(test.s)))
			require.Equal(t, test.want, got)

			var labels []label
			require.NoError(t, Array(&labels).Scan([]byte(test.s)))
			require.Len(t, labels, len(test.want))
			for i, want := range test.want {
				require.Equal(t, label(want), labels[i])
			}

			var ptrs []*string
			require.NoError(t, Array(&ptrs).Scan([]byte(test.s)))
			require.Len(t, ptrs, len(test.want))
			for i, want := range test.want {
				require.NotNil(t, ptrs[i])
				require.Equal(t, want, *ptrs[i])
			}
		})
	}
}
