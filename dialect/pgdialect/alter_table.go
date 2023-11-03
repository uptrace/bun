package pgdialect

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
)

func (d *Dialect) Migrator(db *bun.DB) sqlschema.Migrator {
	return &Migrator{db: db, BaseMigrator: sqlschema.NewBaseMigrator(db)}
}

type Migrator struct {
	*sqlschema.BaseMigrator

	db *bun.DB
}

func (m *Migrator) RenameTable(ctx context.Context, oldName, newName string) error {
	query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", oldName, newName)
	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
