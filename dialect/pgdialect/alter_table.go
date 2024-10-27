package pgdialect

import (
	"context"
	"fmt"
	"log"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

func (d *Dialect) Migrator(db *bun.DB) sqlschema.Migrator {
	return &migrator{db: db, BaseMigrator: sqlschema.NewBaseMigrator(db)}
}

type migrator struct {
	*sqlschema.BaseMigrator

	db *bun.DB
}

var _ sqlschema.Migrator = (*migrator)(nil)

func (m *migrator) Apply(ctx context.Context, changes ...interface{}) error {
	if len(changes) == 0 {
		return nil
	}
	var conn bun.IConn
	var err error

	if conn, err = m.db.Conn(ctx); err != nil {
		return err
	}

	fmter := m.db.Formatter()
	for _, change := range changes {
		var b []byte // TODO(dyma): call db.MakeQueryBytes

		switch change := change.(type) {
		case *migrate.CreateTable:
			log.Printf("create table %q", change.FQN.Table)
			err = m.CreateTable(ctx, change.Model)
			if err != nil {
				return fmt.Errorf("apply changes: create table %s: %w", change.FQN, err)
			}
			continue
		case *migrate.DropTable:
			log.Printf("drop table %q", change.FQN.Table)
			err = m.DropTable(ctx, change.FQN)
			if err != nil {
				return fmt.Errorf("apply changes: drop table %s: %w", change.FQN, err)
			}
			continue
		case *migrate.RenameTable:
			b, err = m.renameTable(fmter, b, change)
		case *migrate.RenameColumn:
			b, err = m.renameColumn(fmter, b, change)
		case *migrate.AddColumn:
			b, err = m.addColumn(fmter, b, change)
		case *migrate.DropColumn:
			b, err = m.dropColumn(fmter, b, change)
		case *migrate.AddForeignKey:
			b, err = m.addForeignKey(fmter, b, change)
		case *migrate.AddUniqueConstraint:
			b, err = m.addUnique(fmter, b, change)
		case *migrate.DropUniqueConstraint:
			b, err = m.dropConstraint(fmter, b, change.FQN, change.Unique.Name)
		case *migrate.DropConstraint:
			b, err = m.dropConstraint(fmter, b, change.FQN(), change.ConstraintName)
		case *migrate.RenameConstraint:
			b, err = m.renameConstraint(fmter, b, change)
		case *migrate.ChangeColumnType:
			b, err = m.changeColumnType(fmter, b, change)
		default:
			return fmt.Errorf("apply changes: unknown operation %T", change)
		}
		if err != nil {
			return fmt.Errorf("apply changes: %w", err)
		}

		query := internal.String(b)
		log.Println("exec query: " + query)
		if _, err = conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("apply changes: %w", err)
		}
	}
	return nil
}

func (m *migrator) renameTable(fmter schema.Formatter, b []byte, rename *migrate.RenameTable) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	b, _ = rename.FQN.AppendQuery(fmter, b)

	b = append(b, " RENAME TO "...)
	b = fmter.AppendName(b, rename.NewName)
	return b, nil
}

func (m *migrator) renameColumn(fmter schema.Formatter, b []byte, rename *migrate.RenameColumn) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	b, _ = rename.FQN.AppendQuery(fmter, b)

	b = append(b, " RENAME COLUMN "...)
	b = fmter.AppendName(b, rename.OldName)

	b = append(b, " TO "...)
	b = fmter.AppendName(b, rename.NewName)

	return b, nil
}

func (m *migrator) addColumn(fmter schema.Formatter, b []byte, add *migrate.AddColumn) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	b, _ = add.FQN.AppendQuery(fmter, b)

	b = append(b, " ADD COLUMN "...)
	b = fmter.AppendName(b, add.Column)
	b = append(b, " "...)

	b, _ = add.ColDef.AppendQuery(fmter, b)

	if add.ColDef.IsIdentity {
		b = appendGeneratedAsIdentity(b)
	}

	return b, nil
}

func (m *migrator) dropColumn(fmter schema.Formatter, b []byte, drop *migrate.DropColumn) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	b, _ = drop.FQN.AppendQuery(fmter, b)

	b = append(b, " DROP COLUMN "...)
	b = fmter.AppendName(b, drop.Column)

	return b, nil
}

func (m *migrator) renameConstraint(fmter schema.Formatter, b []byte, rename *migrate.RenameConstraint) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	fqn := rename.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " RENAME CONSTRAINT "...)
	b = fmter.AppendName(b, rename.OldName)

	b = append(b, " TO "...)
	b = fmter.AppendName(b, rename.NewName)

	return b, nil
}

func (m *migrator) addUnique(fmter schema.Formatter, b []byte, change *migrate.AddUniqueConstraint) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	if b, err = change.FQN.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " ADD CONSTRAINT "...)
	if change.Unique.Name != "" {
		b = fmter.AppendName(b, change.Unique.Name)
	} else {
		// Default naming scheme for unique constraints in Postgres is <table>_<column>_key
		b = fmter.AppendName(b, fmt.Sprintf("%s_%s_key", change.FQN.Table, change.Unique.Columns))
	}
	b = append(b, " UNIQUE ("...)
	b, _ = change.Unique.Columns.Safe().AppendQuery(fmter, b)
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) dropConstraint(fmter schema.Formatter, b []byte, fqn schema.FQN, name string) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " DROP CONSTRAINT "...)
	b = fmter.AppendName(b, name)

	return b, nil
}

func (m *migrator) addForeignKey(fmter schema.Formatter, b []byte, add *migrate.AddForeignKey) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	fqn := add.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " ADD CONSTRAINT "...)
	b = fmter.AppendName(b, add.ConstraintName)

	b = append(b, " FOREIGN KEY ("...)
	if b, err = add.FK.From.Column.Safe().AppendQuery(fmter, b); err != nil {
		return b, err
	}
	b = append(b, ") "...)

	other := schema.FQN{Schema: add.FK.To.Schema, Table: add.FK.To.Table}
	b = append(b, " REFERENCES "...)
	if b, err = other.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " ("...)
	if b, err = add.FK.To.Column.Safe().AppendQuery(fmter, b); err != nil {
		return b, err
	}
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) changeColumnType(fmter schema.Formatter, b []byte, colDef *migrate.ChangeColumnType) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	b, _ = colDef.FQN.AppendQuery(fmter, b)

	// alterColumn never re-assigns err, so there is no need to check for err != nil after calling it
	var i int
	appendAlterColumn := func() {
		if i > 0 {
			b = append(b, ","...)
		}
		b = append(b, " ALTER COLUMN "...)
		b = fmter.AppendName(b, colDef.Column)
		i++
	}

	got, want := colDef.From, colDef.To

	if want.SQLType != got.SQLType {
		appendAlterColumn()
		b = append(b, " SET DATA TYPE "...)
		if b, err = want.AppendQuery(fmter, b); err != nil {
			return b, err
		}
	}

	// Column must be declared NOT NULL before identity can be added.
	// Although PG can resolve the order of operations itself, we make this explicit in the query.
	if want.IsNullable != got.IsNullable {
		appendAlterColumn()
		if !want.IsNullable {
			b = append(b, " SET NOT NULL"...)
		} else {
			b = append(b, " DROP NOT NULL"...)
		}
	}

	if want.IsIdentity != got.IsIdentity {
		appendAlterColumn()
		if !want.IsIdentity {
			b = append(b, " DROP IDENTITY"...)
		} else {
			b = append(b, " ADD"...)
			b = appendGeneratedAsIdentity(b)
		}
	}

	if want.DefaultValue != got.DefaultValue {
		appendAlterColumn()
		if want.DefaultValue == "" {
			b = append(b, " DROP DEFAULT"...)
		} else {
			b = append(b, " SET DEFAULT "...)
			b = append(b, want.DefaultValue...)
		}
	}

	return b, nil
}
