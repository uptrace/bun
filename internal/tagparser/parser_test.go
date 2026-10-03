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
	{"id,pk,type:tinyint;pg=smallint;mssql=tinyint,notnull", "id", map[string][]string{"pk": {""}, "type": {"tinyint;pg=smallint;mssql=tinyint"}, "notnull": {""}}},
	{"type:decimal(10,2);pg=numeric(12,4),notnull", "", map[string][]string{"type": {"decimal(10,2);pg=numeric(12,4)"}, "notnull": {""}}},
	{"type:enum('a;b','x=y');pg=text,default:'x=y'", "", map[string][]string{"type": {"enum('a;b','x=y');pg=text"}, "default": {"'x=y'"}}},
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
