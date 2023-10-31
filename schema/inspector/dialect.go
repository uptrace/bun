package inspector

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type Dialect interface {
	schema.Dialect
	Inspector(db *bun.DB, excludeTables ...string) schema.Inspector
}
