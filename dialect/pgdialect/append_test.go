package pgdialect

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/schema"
)

func TestHStoreAppender(t *testing.T) {
	tests := []struct {
		input      map[string]string
		expectedIn []string // maps being unsorted, multiple expected output are valid
	}{
		{nil, []string{`NULL`}},
		{map[string]string{}, []string{`''`}},

		{map[string]string{"": ""}, []string{`'""=>""'`}},
		{map[string]string{`\`: `\`}, []string{`'"\\"=>"\\"'`}},
		{map[string]string{"'": "'"}, []string{`'"''"=>"''"'`}},
		{map[string]string{`'"{}`: `'"{}`}, []string{`'"''\"{}"=>"''\"{}"'`}},

		{map[string]string{"1": "2", "3": "4"}, []string{`'"1"=>"2","3"=>"4"'`, `'"3"=>"4","1"=>"2"'`}},
		{map[string]string{"1": ""}, []string{`'"1"=>""'`}},
		{map[string]string{"1": "NULL"}, []string{`'"1"=>"NULL"'`}},
		{map[string]string{"{1}": "{2}", "{3}": "{4}"}, []string{`'"{1}"=>"{2}","{3}"=>"{4}"'`, `'"{3}"=>"{4}","{1}"=>"{2}"'`}},
	}

	appendFunc := pgDialect.hstoreAppender(reflect.TypeFor[map[string]string]())

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got := appendFunc(schema.NewFormatter(pgDialect), []byte{}, reflect.ValueOf(test.input))
			require.Contains(t, test.expectedIn, string(got))
		})
	}
}
