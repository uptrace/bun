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

var _ sqlschema.Migrator = (*Migrator)(nil)

func (m *Migrator) RenameTable(ctx context.Context, oldName, newName string) error {
	query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", oldName, newName)
	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

func (m *Migrator) AddContraint(ctx context.Context, fk sqlschema.FK, name string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? ADD CONSTRAINT ? FOREIGN KEY (?) REFERENCES ?.? (?)",
		bun.Safe(fk.From.Schema), bun.Safe(fk.From.Table), bun.Safe(name),
		bun.Safe(fk.From.Column.String()),
		bun.Safe(fk.To.Schema), bun.Safe(fk.To.Table),
		bun.Safe(fk.To.Column.String()),
	)
	if _, err := q.Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) DropContraint(ctx context.Context, schema, table, name string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? DROP CONSTRAINT ?",
		bun.Safe(schema), bun.Safe(table), bun.Safe(name),
	)
	if _, err := q.Exec(ctx); err != nil {
		return err
	}
	return nil
}
