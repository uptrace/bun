package bun

import (
	"database/sql"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"weak"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

const pgMaxIdentifierLen = 63

var relationAliasPattern = regexp.MustCompile(`(?i)bun_alias_[0-9]+`)
var relationUnicodeIdentifierPattern = regexp.MustCompile(`(?i)u&"((?:""|[^"])*)"`)

// Weak keys let mappings follow result sets without retaining sql.Rows.
var rowsColumnAliases sync.Map // weak.Pointer[sql.Rows] -> map[string]string

// relationAliases belongs to one generated SELECT, never to the cached schema.
// minimal-code: aliases are numbered per query; use short relation tags when handwritten SQL needs stable JOIN names.
type relationAliases struct {
	names map[string]string
	used  map[string]string
	next  int
	spans []relationAliasSpan
}

type relationAliasSpan struct {
	start, end int
	name       string
}

func (q *SelectQuery) newRelationAliases(gen schema.QueryGen) *relationAliases {
	if gen.Dialect().Name() != dialect.PG || q.tableModel == nil {
		return nil
	}

	var needed bool
	_ = q.forEachInlineRelJoin(func(j *relationJoin) error {
		n := relationAliasLen(j)
		if n > pgMaxIdentifierLen {
			needed = true
		}
		for _, field := range j.JoinModel.Table().Fields {
			if n+2+len(field.Name) > pgMaxIdentifierLen {
				needed = true
			}
		}
		return nil
	})
	if !needed {
		return nil
	}

	a := new(relationAliases)
	// Reserve short names first so shortening never changes an existing alias.
	if q.table != nil {
		a.reserve(q.table.Alias)
		for _, field := range q.table.Fields {
			a.reserve(field.Name)
		}
	}
	_ = q.forEachInlineRelJoin(func(j *relationJoin) error {
		path := string(appendAlias(nil, j))
		a.reserve(path)
		for _, field := range j.JoinModel.Table().Fields {
			a.reserve(path + "__" + field.Name)
		}
		return nil
	})
	return a
}

func relationAliasLen(j *relationJoin) int {
	n := len(j.Relation.Field.Name)
	if j.hasParent() {
		n += relationAliasLen(j.Parent) + 2
	}
	return n
}

func (a *relationAliases) reserve(name string) {
	if len(name) > pgMaxIdentifierLen {
		return
	}
	if a.used == nil {
		a.used = make(map[string]string)
	}
	a.used[name] = name
}

func (a *relationAliases) append(b []byte, start int) []byte {
	if len(b)-start <= pgMaxIdentifierLen {
		return b
	}
	name := string(b[start:])
	b = append(b[:start], a.alias(name)...)
	a.spans = append(a.spans, relationAliasSpan{start: start, end: len(b), name: name})
	return b
}

func (a *relationAliases) alias(name string) string {
	if alias, ok := a.names[name]; ok {
		return alias
	}
	for {
		alias := "bun_alias_" + strconv.Itoa(a.next)
		a.next++
		if _, ok := a.used[alias]; ok {
			continue
		}
		if a.names == nil {
			a.names = make(map[string]string)
		}
		if a.used == nil {
			a.used = make(map[string]string)
		}
		a.names[name] = alias
		a.used[alias] = name
		return alias
	}
}

// Inspect formatted user SQL once, excluding the aliases Bun generated itself.
// Conservatively reserving names in literals and comments is harmless.
func (a *relationAliases) avoidCollisions(b []byte) []byte {
	if len(a.spans) == 0 {
		return b
	}
	var end int
	for _, span := range a.spans {
		a.reserveSQL(b[end:span.start])
		end = span.end
	}
	a.reserveSQL(b[end:])

	var changed bool
	for _, span := range a.spans {
		alias := a.names[span.name]
		if a.used[alias] != span.name {
			delete(a.names, span.name)
			a.alias(span.name)
			changed = true
		}
	}
	if !changed {
		return b
	}

	out := make([]byte, 0, len(b))
	end = 0
	for _, span := range a.spans {
		out = append(out, b[end:span.start]...)
		out = append(out, a.names[span.name]...)
		end = span.end
	}
	return append(out, b[end:]...)
}

func (a *relationAliases) reserveSQL(b []byte) {
	a.reserveNames(b)
	for _, match := range relationUnicodeIdentifierPattern.FindAllSubmatch(b, -1) {
		name := string(match[1])
		// Any UESCAPE character used in this identifier must occur in its body.
		// Trying each present character avoids parsing the surrounding SQL grammar.
		seen := make(map[rune]bool)
		for _, escape := range name {
			if !seen[escape] {
				seen[escape] = true
				a.reserveNames([]byte(unescapeRelationIdentifier(name, string(escape))))
			}
		}
	}
}

func (a *relationAliases) reserveNames(b []byte) {
	for _, name := range relationAliasPattern.FindAll(b, -1) {
		a.reserve(strings.ToLower(string(name)))
	}
}

func unescapeRelationIdentifier(name, escape string) string {
	var out strings.Builder
	for {
		i := strings.Index(name, escape)
		if i < 0 {
			out.WriteString(name)
			return out.String()
		}
		out.WriteString(name[:i])
		name = name[i+len(escape):]
		if strings.HasPrefix(name, escape) {
			out.WriteString(escape)
			name = name[len(escape):]
			continue
		}
		n := 4
		if strings.HasPrefix(name, "+") {
			n = 6
			name = name[1:]
		}
		if len(name) < n {
			return ""
		}
		value, err := strconv.ParseUint(name[:n], 16, 32)
		if err != nil {
			return ""
		}
		out.WriteRune(rune(value))
		name = name[n:]
	}
}

func rememberRowsColumnAliases(rows *sql.Rows, aliases map[string]string) {
	if len(aliases) == 0 {
		return
	}
	key := weak.Make(rows)
	rowsColumnAliases.Store(key, aliases)
	runtime.AddCleanup(rows, func(key weak.Pointer[sql.Rows]) {
		rowsColumnAliases.Delete(key)
	}, key)
}

func scanRowsColumnAliases(rows *sql.Rows, model Model, consume bool) {
	key := weak.Make(rows)
	var aliases any
	var ok bool
	if consume {
		aliases, ok = rowsColumnAliases.LoadAndDelete(key)
	} else {
		aliases, ok = rowsColumnAliases.Load(key)
	}
	if ok {
		setColumnAliases(model, aliases.(map[string]string))
	}
}

func setColumnAliases(model Model, aliases map[string]string) {
	if model, ok := model.(interface{ setColumnAliases(map[string]string) }); ok {
		model.setColumnAliases(aliases)
	}
}

// Restore once per result set, before scanning any rows or mounting joined models.
func restoreColumnAliases(columns []string, aliases map[string]string) []string {
	if len(aliases) == 0 {
		return columns
	}
	for i, column := range columns {
		if original, ok := aliases[unquote(column)]; ok {
			columns[i] = original
		}
	}
	return columns
}
