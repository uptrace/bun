package bun

import (
	"context"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type (
	// Safe marks a SQL fragment as trusted and prevents further escaping.
	Safe = schema.Safe
	// Name represents a SQL identifier such as a column or table name.
	Name = schema.Name
	// Ident is a fully qualified SQL identifier.
	Ident = schema.Ident
	// Order denotes sorting direction used in ORDER BY clauses.
	Order = schema.Order

	// NullTime is a nullable time value compatible with Bun.
	NullTime = schema.NullTime
	// BaseModel provides default metadata embedded into user models.
	BaseModel = schema.BaseModel
	// Query is implemented by all Bun query builders.
	Query = schema.Query

	// BeforeAppendModelHook is called before a model is appended to a query.
	BeforeAppendModelHook = schema.BeforeAppendModelHook

	// BeforeScanRowHook runs before scanning an individual row.
	BeforeScanRowHook = schema.BeforeScanRowHook
	// AfterScanRowHook runs after scanning an individual row.
	AfterScanRowHook = schema.AfterScanRowHook
)

const (
	// OrderAsc sorts values in ascending order.
	OrderAsc = schema.OrderAsc
	// OrderAscNullsFirst sorts ascending with NULL values first.
	OrderAscNullsFirst = schema.OrderDesc
	// OrderAscNullsLast sorts ascending with NULL values last.
	OrderAscNullsLast = schema.OrderAscNullsLast
	// OrderDesc sorts values in descending order.
	OrderDesc = schema.OrderDesc
	// OrderDescNullsFirst sorts descending with NULL values first.
	OrderDescNullsFirst = schema.OrderDescNullsFirst
	// OrderDescNullsLast sorts descending with NULL values last.
	OrderDescNullsLast = schema.OrderDescNullsLast
)

// SafeQuery wraps a raw query string and arguments and marks it safe for Bun.
func SafeQuery(query string, args ...any) schema.QueryWithArgs {
	return schema.SafeQuery(query, args)
}

// BeforeSelectHook is invoked before executing SELECT queries.
type BeforeSelectHook interface {
	BeforeSelect(ctx context.Context, query *SelectQuery) error
}

// AfterSelectHook is invoked after executing SELECT queries.
type AfterSelectHook interface {
	AfterSelect(ctx context.Context, query *SelectQuery) error
}

// BeforeInsertHook is invoked before executing INSERT queries.
type BeforeInsertHook interface {
	BeforeInsert(ctx context.Context, query *InsertQuery) error
}

// AfterInsertHook is invoked after executing INSERT queries.
type AfterInsertHook interface {
	AfterInsert(ctx context.Context, query *InsertQuery) error
}

// BeforeUpdateHook is invoked before executing UPDATE queries.
type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context, query *UpdateQuery) error
}

// AfterUpdateHook is invoked after executing UPDATE queries.
type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context, query *UpdateQuery) error
}

// BeforeDeleteHook is invoked before executing DELETE queries.
type BeforeDeleteHook interface {
	BeforeDelete(ctx context.Context, query *DeleteQuery) error
}

// AfterDeleteHook is invoked after executing DELETE queries.
type AfterDeleteHook interface {
	AfterDelete(ctx context.Context, query *DeleteQuery) error
}

// BeforeCreateTableHook is invoked before executing CREATE TABLE queries.
type BeforeCreateTableHook interface {
	BeforeCreateTable(ctx context.Context, query *CreateTableQuery) error
}

// AfterCreateTableHook is invoked after executing CREATE TABLE queries.
type AfterCreateTableHook interface {
	AfterCreateTable(ctx context.Context, query *CreateTableQuery) error
}

// BeforeDropTableHook is invoked before executing DROP TABLE queries.
type BeforeDropTableHook interface {
	BeforeDropTable(ctx context.Context, query *DropTableQuery) error
}

// AfterDropTableHook is invoked after executing DROP TABLE queries.
type AfterDropTableHook interface {
	AfterDropTable(ctx context.Context, query *DropTableQuery) error
}

// SetLogger overwrites default Bun logger.
func SetLogger(logger internal.Logging) {
	internal.SetLogger(logger)
}

// In wraps a slice so it can be used with the IN clause.
func In(slice any) schema.QueryAppender {
	return schema.In(slice)
}

// NullZero forces zero values to be treated as NULL when building queries.
func NullZero(value any) schema.QueryAppender {
	return schema.NullZero(value)
}
