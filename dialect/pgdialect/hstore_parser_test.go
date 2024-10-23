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
