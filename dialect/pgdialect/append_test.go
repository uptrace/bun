package pgdialect

import (
	"fmt"
	"math"
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
			got := appendFunc(schema.NewQueryGen(pgDialect), []byte{}, reflect.ValueOf(test.input))
			require.Contains(t, test.expectedIn, string(got))
		})
	}
}

func TestAppendUintAsInt(t *testing.T) {
	gen := schema.NewQueryGen(New(WithAppendUintAsInt(true)))

	require.Equal(t, "-1", gen.FormatQuery("?", uint64(math.MaxUint64)))
	// uint takes the same path as uint64.
	u := uint(math.MaxUint)
	require.Equal(t, gen.FormatQuery("?", uint64(u)), gen.FormatQuery("?", u))
	// so do model fields and defined types.
	type Hash uint
	require.Equal(t, gen.FormatQuery("?", uint64(u)), gen.FormatQuery("?", Hash(u)))
	require.Equal(t, gen.FormatQuery("?", uint64(u)), gen.FormatQuery("?", &u))

	// a negative result after '-' must not start a line comment.
	require.Equal(t, "1000- -1", gen.FormatQuery("1000-?", uint64(math.MaxUint64)))
	require.Equal(t, "1000- -1", gen.FormatQuery("1000-?", uint32(math.MaxUint32)))
	require.Equal(t, "1000-1", gen.FormatQuery("1000-?", uint64(1)))
}
