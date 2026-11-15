package pgdialect

import (
	"encoding/json"
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

func TestArrayScanJSONStrings(t *testing.T) {
	for _, value := range []string{
		"plain", "}", "{", "{nested}", `"}`, `"{`, `\`, `\"}`, `\\`,
	} {
		t.Run(value, func(t *testing.T) {
			want := []map[string]string{{"text": value}, {"text": "last"}}
			data, err := json.Marshal(want)
			require.NoError(t, err)

			var got []map[string]string
			require.NoError(t, Array(&got).Scan(data))
			require.Equal(t, want, got)
		})
	}
}
