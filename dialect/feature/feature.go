package feature

import (
	"fmt"
	"strconv"

	"github.com/uptrace/bun/internal"
)

type Feature = internal.Flag

const (
	CTE Feature = 1 << iota
	WithValues
	Returning
	InsertReturning
	Output // mssql
	DefaultPlaceholder
	DoubleColonCast
	ValuesRow
	UpdateMultiTable
	InsertTableAlias
	UpdateTableAlias
	DeleteTableAlias
	AutoIncrement
	Identity
	TableCascade
	TableIdentity
	TableTruncate
	InsertOnConflict     // INSERT ... ON CONFLICT
	InsertOnDuplicateKey // INSERT ... ON DUPLICATE KEY
	InsertIgnore         // INSERT IGNORE ...
	TableNotExists
	OffsetFetch
	SelectExists
	UpdateFromTable
	MSSavepoint
	GeneratedIdentity
	CompositeIn      // ... WHERE (A,B) IN ((N, NN), (N, NN)...)
	UpdateOrderLimit // UPDATE ... ORDER BY ... LIMIT ...
	DeleteOrderLimit // DELETE ... ORDER BY ... LIMIT ...
	DeleteReturning
	AlterColumnExists // ADD/DROP COLUMN IF NOT EXISTS/IF EXISTS
)

type NotSupportError struct {
	Flag Feature
}

func (err *NotSupportError) Error() string {
	name, ok := flag2str[err.Flag]
	if !ok {
		name = strconv.FormatInt(int64(err.Flag), 10)
	}
	return fmt.Sprintf("bun: feature %s is not supported by current dialect", name)
}

func NewNotSupportError(flag Feature) *NotSupportError {
	return &NotSupportError{Flag: flag}
}

var flag2str = map[Feature]string{
	CTE:                  "CTE",
	WithValues:           "WithValues",
	Returning:            "Returning",
	InsertReturning:      "InsertReturning",
	Output:               "Output",
	DefaultPlaceholder:   "DefaultPlaceholder",
	DoubleColonCast:      "DoubleColonCast",
	ValuesRow:            "ValuesRow",
	UpdateMultiTable:     "UpdateMultiTable",
	InsertTableAlias:     "InsertTableAlias",
	UpdateTableAlias:     "UpdateTableAlias",
	DeleteTableAlias:     "DeleteTableAlias",
	AutoIncrement:        "AutoIncrement",
	Identity:             "Identity",
	TableCascade:         "TableCascade",
	TableIdentity:        "TableIdentity",
	TableTruncate:        "TableTruncate",
	InsertOnConflict:     "InsertOnConflict",
	InsertOnDuplicateKey: "InsertOnDuplicateKey",
	InsertIgnore:         "InsertIgnore",
	TableNotExists:       "TableNotExists",
	OffsetFetch:          "OffsetFetch",
	SelectExists:         "SelectExists",
	UpdateFromTable:      "UpdateFromTable",
	MSSavepoint:          "MSSavepoint",
	GeneratedIdentity:    "GeneratedIdentity",
	CompositeIn:          "CompositeIn",
	UpdateOrderLimit:     "UpdateOrderLimit",
	DeleteOrderLimit:     "DeleteOrderLimit",
	DeleteReturning:      "DeleteReturning",
	AlterColumnExists:    "AlterColumnExists",
}
