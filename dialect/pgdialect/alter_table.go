package pgdialect

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun/migrate/sqlschema"
)

func (d *Dialect) Migrator(sqldb *sql.DB) sqlschema.Migrator {
	return &Migrator{sqldb: sqldb}
}

type Migrator struct {
	sqldb *sql.DB
}

func (m *Migrator) RenameTable(ctx context.Context, oldName, newName string) error {
	query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", oldName, newName)
	_, err := m.sqldb.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
