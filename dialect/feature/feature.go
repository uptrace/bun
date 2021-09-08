package feature

import "github.com/uptrace/bun/internal"

type Feature = internal.Flag

const (
	CTE Feature = 1 << iota
	Returning
	DefaultPlaceholder
	DoubleColonCast
	ValuesRow
	UpdateMultiTable
	InsertTableAlias
	DeleteTableAlias
	AutoIncrement
	TableCascade
	TableIdentity
	TableTruncate
	OnDuplicateKey
)
