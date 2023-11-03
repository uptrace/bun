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
	CreateTable(ctx context.Context, model interface{}) error
	DropTable(ctx context.Context, schema, table string) error
}

// Migrator is a dialect-agnostic wrapper for sqlschema.Dialect
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

// BaseMigrator can be embeded by dialect's Migrator implementations to re-use some of the existing bun queries.
type BaseMigrator struct {
	db *bun.DB
}

func NewBaseMigrator(db *bun.DB) *BaseMigrator {
	return &BaseMigrator{db: db}
}

func (m *BaseMigrator) CreateTable(ctx context.Context, model interface{}) error {
	_, err := m.db.NewCreateTable().Model(model).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (m *BaseMigrator) DropTable(ctx context.Context, schema, name string) error {
	_, err := m.db.NewDropTable().TableExpr("?.?", bun.Ident(schema), bun.Ident(name)).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}
