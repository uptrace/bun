package feature

import "github.com/uptrace/bun/internal"

type Feature = internal.Flag

const (
	CTE Feature = 1 << iota
	Returning
	InsertReturning
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
	InsertOnConflict     // INSERT ... ON CONFLICT
	InsertOnDuplicateKey // INSERT ... ON DUPLICATE KEY
	InsertIgnore         // INSERT IGNORE ...
)
