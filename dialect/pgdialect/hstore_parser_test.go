package pgdialect

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHStoreParser(t *testing.T) {
	tests := []struct {
		s string
		m map[string]string
	}{
		{``, map[string]string{}},

		{`""=>""`, map[string]string{"": ""}},
		{`"\\"=>"\\"`, map[string]string{`\`: `\`}},
		{`"'"=>"'"`, map[string]string{"'": "'"}},
		{`"'\"{}"=>"'\"{}"`, map[string]string{`'"{}`: `'"{}`}},

		{`"1"=>"2", "3"=>"4"`, map[string]string{"1": "2", "3": "4"}},
		{`"1"=>NULL`, map[string]string{"1": ""}},
		{`"1"=>"NULL"`, map[string]string{"1": "NULL"}},
		{`"{1}"=>"{2}", "{3}"=>"{4}"`, map[string]string{"{1}": "{2}", "{3}": "{4}"}},

		// hstore keys and values are text, so a leading \x is not bytea hex.
		{`"\\x41"=>"\\x4142"`, map[string]string{`\x41`: `\x4142`}},
		{`"1"=>"\\xzz"`, map[string]string{"1": `\xzz`}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			p := newHStoreParser([]byte(test.s))

			got := make(map[string]string)
			for p.Next() {
				got[p.Key()] = p.Value()
			}

			require.NoError(t, p.Err())
			require.Equal(t, test.m, got)
		})
	}
}
