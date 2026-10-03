package mssqldialect

import (
	"testing"

	"github.com/uptrace/bun/schema"
)

func TestAppendJSONUnicode(t *testing.T) {
	const jsonb = `{"k":"日本語"}`

	tests := []struct {
		name    string
		dialect *Dialect
		want    string
	}{
		{name: "unicode", dialect: New(), want: `N'{"k":"日本語"}'`},
		{name: "no unicode", dialect: New(WithUnicode(false)), want: `'{"k":"日本語"}'`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := string(test.dialect.AppendJSON(nil, []byte(jsonb))); got != test.want {
				t.Errorf("AppendJSON = %s, want %s", got, test.want)
			}
			if got := string(test.dialect.AppendString(nil, jsonb)); got != test.want {
				t.Errorf("AppendString = %s, want %s", got, test.want)
			}
		})
	}
}

func TestAppendJSONValueUnicode(t *testing.T) {
	gen := schema.NewQueryGen(New())

	const want = `N'{"k":"日本語"}'`
	if got := string(gen.Append(nil, map[string]string{"k": "日本語"})); got != want {
		t.Errorf("Append = %s, want %s", got, want)
	}
}
