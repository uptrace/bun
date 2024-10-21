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
	Apply(ctx context.Context, changes ...interface{}) error
}

// migrator is a dialect-agnostic wrapper for sqlschema.MigratorDialect.
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

func (m *BaseMigrator) DropTable(ctx context.Context, fqn schema.FQN) error {
	_, err := m.db.NewDropTable().TableExpr(fqn.String()).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}
