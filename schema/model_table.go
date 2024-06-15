package schema

import "github.com/uptrace/bun/internal/tableparser"

type ModelTable QueryWithArgs

func NewModelTable(query string, args ...interface{}) ModelTable {
	return ModelTable(SafeQuery(query, args))
}

func (t ModelTable) IsZero() bool {
	return QueryWithArgs(t).IsZero()
}

func (t ModelTable) HasTableName() bool {
	return !t.IsZero() && t.Query != ""
}

func (t ModelTable) AppendQuery(fmter Formatter, dst []byte) ([]byte, error) {
	return QueryWithArgs(t).AppendQuery(fmter, dst)
}

func (t ModelTable) GetTableName(fmter Formatter) string {
	formattedQuery := fmter.FormatQuery(t.Query, t.Args...)
	table := tableparser.Parse(formattedQuery)
	return table.Name()
}
