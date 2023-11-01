package sqlschema

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type MigratorDialect interface {
	schema.Dialect
	Migrator(*bun.DB) Migrator
}

type Migrator interface {
	RenameTable(ctx context.Context, oldName, newName string) error
}

type migrator struct {
	Migrator
}

func NewMigrator(db *bun.DB) (Migrator, error) {
	md, ok := db.Dialect().(MigratorDialect)
	if !ok {
		return nil, fmt.Errorf("%q dialect does not implement sqlschema.Migrator", db.Dialect().Name())
	}
	return &migrator{
		Migrator: md.Migrator(db),
	}, nil
}
