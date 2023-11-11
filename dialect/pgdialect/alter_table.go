package pgdialect

import (
	"context"

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

func (m *Migrator) exec(ctx context.Context, q *bun.RawQuery) error {
	if _, err := q.Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) RenameTable(ctx context.Context, oldName, newName string) error {
	q := m.db.NewRaw("ALTER TABLE ? RENAME TO ?", bun.Ident(oldName), bun.Ident(newName))
	return m.exec(ctx, q)
}

func (m *Migrator) AddContraint(ctx context.Context, fk sqlschema.FK, name string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? ADD CONSTRAINT ? FOREIGN KEY (?) REFERENCES ?.? (?)",
		bun.Safe(fk.From.Schema), bun.Safe(fk.From.Table), bun.Safe(name),
		bun.Safe(fk.From.Column.String()),
		bun.Safe(fk.To.Schema), bun.Safe(fk.To.Table),
		bun.Safe(fk.To.Column.String()),
	)
	return m.exec(ctx, q)
}

func (m *Migrator) DropContraint(ctx context.Context, schema, table, name string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? DROP CONSTRAINT ?",
		bun.Ident(schema), bun.Ident(table), bun.Ident(name),
	)
	return m.exec(ctx, q)
}

func (m *Migrator) RenameConstraint(ctx context.Context, schema, table, oldName, newName string) error {
	q := m.db.NewRaw(
		"ALTER TABLE ?.? RENAME CONSTRAINT ? TO ?",
		bun.Ident(schema), bun.Ident(table), bun.Ident(oldName), bun.Ident(newName),
	)
	return m.exec(ctx, q)
}
