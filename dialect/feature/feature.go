package feature

import "github.com/uptrace/bun/internal"

type Feature = internal.Flag

const DefaultFeatures = Returning | TableCascade

const (
	Returning Feature = 1 << iota
	DefaultPlaceholder
	DoubleColonCast
	ValuesRow
	UpdateMultiTable
	AutoIncrement
	TableCascade
	TableIdentity
	TableTruncate
	OnDuplicateKey
)
