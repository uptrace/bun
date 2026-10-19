package bun

import (
	"database/sql"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"weak"

	"github.com/stretchr/testify/require"
)

func TestRelationAliasNames(t *testing.T) {
	for _, name := range []string{
		strings.Repeat("a", 63),
		strings.Repeat("a", 64),
		strings.Repeat("a", 256),
		strings.Repeat("界", 32),
	} {
		t.Run(name, func(t *testing.T) {
			var names relationAliases
			prefix := []byte("SELECT ")
			b := names.append(append(prefix, name...), len(prefix))
			alias := string(b[len(prefix):])
			require.LessOrEqual(t, len(alias), pgMaxIdentifierLen)
			require.True(t, utf8.ValidString(alias))
			require.Equal(t, "SELECT ", string(b[:len(prefix)]))
			if len(name) <= pgMaxIdentifierLen {
				require.Equal(t, name, alias)
			} else {
				require.NotEqual(t, name, alias)
				require.Equal(t, alias, string(names.append([]byte(name), 0)))
			}
		})
	}
}

func TestRelationAliasNamesPreserveDistinctSuffixes(t *testing.T) {
	var names relationAliases
	prefix := strings.Repeat("nested_relation__", 8)
	first := string(names.append([]byte(prefix+"display_name"), 0))
	second := string(names.append([]byte(prefix+"display_code"), 0))
	require.NotEqual(t, first, second)
	require.Equal(t, first, string(names.append([]byte(prefix+"display_name"), 0)))
}

func TestRelationAliasNamesAvoidReservedNames(t *testing.T) {
	name := strings.Repeat("nested_relation__", 8) + "display_name"
	var original relationAliases
	collision := string(original.append([]byte(name), 0))

	var names relationAliases
	names.reserve(collision)
	alias := string(names.append([]byte(name), 0))
	require.NotEqual(t, collision, alias)
	require.Equal(t, collision, string(names.append([]byte(collision), 0)))
	require.Equal(t, alias, string(names.append([]byte(name), 0)))
	require.Equal(t, name, names.used[alias])
}

func TestRelationAliasesRepairRepeatedSpans(t *testing.T) {
	var aliases relationAliases
	for i := range 9 {
		aliases.reserve("bun_alias_" + string(rune('0'+i)))
	}
	name := strings.Repeat("relation__", 8) + "name"
	b := []byte("SELECT ")
	start := len(b)
	b = aliases.append(append(b, name...), start)
	b = append(b, " FROM "...)
	start = len(b)
	b = aliases.append(append(b, name...), start)
	b = append(b, `, 7 AS "bun_alias_9", 8 AS "bun_alias_10"`...)

	got := string(aliases.avoidCollisions(b))
	require.Equal(t, `SELECT bun_alias_11 FROM bun_alias_11, 7 AS "bun_alias_9", 8 AS "bun_alias_10"`, got)
	require.Equal(t, name, aliases.used["bun_alias_11"])
	require.Equal(t, "bun_alias_9", aliases.used["bun_alias_9"])
}

func TestRowsColumnAliasesConsumption(t *testing.T) {
	rows := new(sql.Rows)
	aliases := map[string]string{"bun_alias_0": "relation__field"}
	rememberRowsColumnAliases(rows, aliases)

	var first, second, rest mapModel
	scanRowsColumnAliases(rows, &first, false)
	scanRowsColumnAliases(rows, &second, false)
	require.Equal(t, aliases, first.columnAliases)
	require.Equal(t, aliases, second.columnAliases)
	scanRowsColumnAliases(rows, &rest, true)
	require.Equal(t, aliases, rest.columnAliases)
	_, ok := rowsColumnAliases.Load(weak.Make(rows))
	require.False(t, ok)
	runtime.KeepAlive(rows)
}

func TestRowsColumnAliasesDoNotRetainRows(t *testing.T) {
	key := func() weak.Pointer[sql.Rows] {
		rows := new(sql.Rows)
		rememberRowsColumnAliases(rows, map[string]string{"bun_alias_0": "relation__field"})
		return weak.Make(rows)
	}()
	require.Eventually(t, func() bool {
		runtime.GC()
		_, ok := rowsColumnAliases.Load(key)
		return !ok
	}, 5*time.Second, 10*time.Millisecond)
}

func TestUnescapeRelationIdentifier(t *testing.T) {
	for _, tt := range []struct {
		name, escape, want string
	}{
		{`bun_\0061lias_0`, `\`, "bun_alias_0"},
		{`bun_\+000061lias_0`, `\`, "bun_alias_0"},
		{`bun_!0061lias_0`, "!", "bun_alias_0"},
		{`a!!b`, "!", "a!b"},
		{`bun_!xyz`, "!", ""},
	} {
		require.Equal(t, tt.want, unescapeRelationIdentifier(tt.name, tt.escape))
	}
}
