package tagparser_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/internal/tagparser"
)

var tagTests = []struct {
	tag     string
	name    string
	options map[string][]string
}{
	{"", "", nil},
	{"hello", "hello", nil},
	{"hello,world", "hello", map[string][]string{"world": {""}}},
	{`"hello,world'`, "", nil},
	{`"hello:world"`, `hello:world`, nil},
	{",hello", "", map[string][]string{"hello": {""}}},
	{",hello,world", "", map[string][]string{"hello": {""}, "world": {""}}},
	{"hello:", "", map[string][]string{"hello": {""}}},
	{"hello:world", "", map[string][]string{"hello": {"world"}}},
	{"hello:world,foo", "", map[string][]string{"hello": {"world"}, "foo": {""}}},
	{"hello:world,foo:bar", "", map[string][]string{"hello": {"world"}, "foo": {"bar"}}},
	{"hello:\"world1,world2\"", "", map[string][]string{"hello": {"world1,world2"}}},
	{`hello:"world1,world2",world3`, "", map[string][]string{"hello": {"world1,world2"}, "world3": {""}}},
	{`hello:"world1:world2",world3`, "", map[string][]string{"hello": {"world1:world2"}, "world3": {""}}},
	{`hello:"D'Angelo, esquire",foo:bar`, "", map[string][]string{"hello": {"D'Angelo, esquire"}, "foo": {"bar"}}},
	{`hello:"world('foo', 'bar')"`, "", map[string][]string{"hello": {"world('foo', 'bar')"}}},
	{" hello,foo: bar ", " hello", map[string][]string{"foo": {" bar "}}},
	{"foo:bar(hello, world)", "", map[string][]string{"foo": {"bar(hello, world)"}}},
	{"foo:bar(hello(), world)", "", map[string][]string{"foo": {"bar(hello(), world)"}}},
	{"type:geometry(POINT, 4326)", "", map[string][]string{"type": {"geometry(POINT, 4326)"}}},
	{"id,pk,type:tinyint;pg=smallint;mssql=tinyint,notnull", "id", map[string][]string{"pk": {""}, "type": {"tinyint"}, "notnull": {""}}},
	{"type:decimal(10,2);pg=numeric(12,4),notnull", "", map[string][]string{"type": {"decimal(10,2)"}, "notnull": {""}}},
	{"type:enum('a;b','x=y');pg=text,default:'x=y'", "", map[string][]string{"type": {"enum('a;b','x=y')"}, "default": {"'x=y'"}}},
	{"rel:belongs-to,join:profile_id=id", "", map[string][]string{"rel": {"belongs-to"}, "join": {"profile_id=id"}}},
	{"foo:bar,foo:baz", "", map[string][]string{"foo": []string{"bar", "baz"}}},
}

func TestTagParser(t *testing.T) {
	for i, test := range tagTests {
		tag := tagparser.Parse(test.tag)
		require.Equal(t, test.name, tag.Name, "#%d", i)
		require.Equal(t, test.options, tag.Options, "#%d", i)
	}
}

func TestOptionOverridesGenericValue(t *testing.T) {
	tag := tagparser.Parse(`format:plain;html=rich;json=compact,join:profile_id=id`)
	value, ok := tag.Option("format")
	require.True(t, ok)
	require.Equal(t, "plain", value)
	join, ok := tag.Option("join")
	require.True(t, ok)
	require.Equal(t, "profile_id=id", join)
}

func TestOptionOverrides(t *testing.T) {
	tests := []struct {
		input     string
		option    string
		value     string
		overrides map[string]string
	}{
		{`format:plain;html=rich;json=compact`, "format", "plain", map[string]string{"html": "rich", "json": "compact"}},
		{`format:plain;custom=value=with=equals;empty=`, "format", "plain", map[string]string{"custom": "value=with=equals", "empty": ""}},
		{`format:plain; html = rich ;html=latest;`, "format", "plain", map[string]string{"html": "latest"}},
		{`type:decimal(10,2);pg=numeric(12,4)`, "type", "decimal(10,2)", map[string]string{"pg": "numeric(12,4)"}},
		{`type:enum('a;b','x=y');pg=text`, "type", "enum('a;b','x=y')", map[string]string{"pg": "text"}},
		{`type:text;pg="a;b=c",notnull`, "type", "text", map[string]string{"pg": `"a;b=c"`}},
		{`format:fn(')',nested(a;b));custom=fn('a,b;c=d')`, "format", "fn(')',nested(a;b))", map[string]string{"custom": "fn('a,b;c=d')"}},
		{`default:'a,b;c=d',notnull`, "default", "'a,b;c=d'", nil},
		{`format:"a,b;c=d";custom=other`, "format", "a,b;c=d", map[string]string{"custom": "other"}},
		{`format:plain;custom='a\';b,c=d'`, "format", "plain", map[string]string{"custom": `'a\';b,c=d'`}},
		{`format:plain;custom='a'';b,c=d'`, "format", "plain", map[string]string{"custom": `'a'';b,c=d'`}},
		{`format:old;html=rich,format:new;json=compact`, "format", "new", map[string]string{"json": "compact"}},
		{`format:old;html=rich,format:new`, "format", "new", nil},
		{`type:"my_schema"."my_type";pg=text`, "type", `"my_schema"."my_type"`, map[string]string{"pg": "text"}},
		{`type:"char"[];pg=text`, "type", `"char"[]`, map[string]string{"pg": "text"}},
		{`format:a)b;html=c)d,next:x`, "format", "a)b", map[string]string{"html": "c)d"}},
		{`format:;custom=other`, "format", "", map[string]string{"custom": "other"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tag := tagparser.Parse(tt.input)
			value, ok := tag.Option(tt.option)
			require.True(t, ok)
			require.Equal(t, tt.value, value)
			require.Equal(t, tt.overrides, tag.Overrides[tt.option])
		})
	}
	t.Run("separate options", func(t *testing.T) {
		tag := tagparser.Parse(`format:plain;html=rich,encoding:utf8;html=ascii`)
		require.Equal(t, "rich", tag.Overrides["format"]["html"])
		require.Equal(t, "ascii", tag.Overrides["encoding"]["html"])
	})
	t.Run("malformed override", func(t *testing.T) {
		require.Panics(t, func() { tagparser.Parse(`format:plain;html`) })
	})
}

func TestOptionClosingParenthesis(t *testing.T) {
	tag := tagparser.Parse(`format:a)b,next:x`)
	value, ok := tag.Option("next")
	require.True(t, ok)
	require.Equal(t, "x", value)
}
