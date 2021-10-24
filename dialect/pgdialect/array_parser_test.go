package pgdialect

import (
	"io"
	"testing"
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
	}

	for testi, test := range tests {
		p := newArrayParser([]byte(test.s))

		var got []string
		for {
			s, err := p.NextElem()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}
			got = append(got, string(s))
		}

		if len(got) != len(test.els) {
			t.Fatalf(
				"test #%d got %d elements, wanted %d (got=%#v wanted=%#v)",
				testi, len(got), len(test.els), got, test.els)
		}

		for i, el := range got {
			if el != test.els[i] {
				t.Fatalf(
					"test #%d el #%d does not match: %s != %s (got=%#v wanted=%#v)",
					testi, i, el, test.els[i], got, test.els)
			}
		}
	}
}
