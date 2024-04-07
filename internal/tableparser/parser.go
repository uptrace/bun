package tableparser

import "strings"

type Table struct {
	name  string
	alias string
}

func (t Table) IsZero() bool {
	return t.name == "" && t.alias == ""
}

func (t Table) Name() string {
	if t.alias != "" {
		return t.alias
	}
	return t.name
}

func Parse(query string) Table {
	if query == "" {
		return Table{}
	}
	cleanedQuery := cleanQuery(query)
	table := parseQuery(cleanedQuery)
	return table
}

func cleanQuery(query string) string {
	// Trim the query to remove any leading or trailing spaces.
	trimmedQuery := strings.TrimSpace(query)

	// Lowercase the keyword AS
	cleanedQuery := strings.ReplaceAll(trimmedQuery, " AS ", " as ")

	return cleanedQuery
}

func parseQuery(query string) Table {
	if hasAlias(query) {
		return parseWithAlias(query)
	}
	return Table{
		name: query,
	}
}

func hasAlias(query string) bool {
	return strings.Contains(query, " as ") ||
		strings.Contains(query, " ")
}

func parseWithAlias(query string) Table {
	if strings.Contains(query, " as ") {
		sp := strings.Split(query, " as ")
		return Table{
			name:  strings.TrimSpace(sp[0]),
			alias: strings.TrimSpace(sp[1]),
		}
	}
	sp := strings.Split(query, " ")
	return Table{
		name:  strings.TrimSpace(sp[0]),
		alias: strings.TrimSpace(sp[1]),
	}
}
