package pgdialect

import (
	"io"
	"testing"
)

func TestHStoreParser(t *testing.T) {
	tests := []struct {
		s string
		m map[string]string
	}{
		{`""=>""`, map[string]string{"": ""}},
		{`"\\"=>"\\"`, map[string]string{`\`: `\`}},
		{`"'"=>"'"`, map[string]string{"'": "'"}},
		{`"'\"{}"=>"'\"{}"`, map[string]string{`'"{}`: `'"{}`}},

		{`"1"=>"2", "3"=>"4"`, map[string]string{"1": "2", "3": "4"}},
		{`"1"=>NULL`, map[string]string{"1": ""}},
		{`"1"=>"NULL"`, map[string]string{"1": "NULL"}},
		{`"{1}"=>"{2}", "{3}"=>"{4}"`, map[string]string{"{1}": "{2}", "{3}": "{4}"}},
	}

	for testi, test := range tests {
		p := newHStoreParser([]byte(test.s))

		got := make(map[string]string)
		for {
			key, err := p.NextKey()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}

			value, err := p.NextValue()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}

			got[key] = value
		}

		if len(got) != len(test.m) {
			t.Fatalf(
				"test #%d got %d elements, wanted %d (got=%#v wanted=%#v)",
				testi, len(got), len(test.m), got, test.m)
		}

		for k, v := range got {
			if v != test.m[k] {
				t.Fatalf(
					"test #%d key #%s does not match: %s != %s (got=%#v wanted=%#v)",
					testi, k, v, test.m[k], got, test.m)
			}
		}
	}
}
