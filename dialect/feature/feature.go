package feature

import (
	"errors"

	"github.com/uptrace/bun/internal"
)

type Feature = internal.Flag

var ErrUnsupportedFeature = errors.New("bun: UnsupportedFeature")

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
	IndexHash
)
