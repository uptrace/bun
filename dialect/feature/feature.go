package feature

import "github.com/uptrace/bun/internal"

type Feature = internal.Flag

const DefaultFeatures = Returning | DropTableCascade

const (
	Returning Feature = 1 << iota
	DefaultPlaceholder
	ValuesRow
	Backticks
	AutoIncrement
	DropTableCascade
)
