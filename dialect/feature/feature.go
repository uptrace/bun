package feature

import "github.com/uptrace/bun/internal"

type Feature = internal.Flag

const DefaultFeatures = Returning | TableCascade

const (
	Returning Feature = 1 << iota
	DefaultPlaceholder
	ValuesRow
	AutoIncrement
	TableCascade
	TableIdentity
	TableTruncate
)
