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

func (m *migrator) Apply(ctx context.Context, changes ...sqlschema.Operation) error {
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
			err = m.CreateTable(ctx, change.Model)
			if err != nil {
				return fmt.Errorf("apply changes: create table %s: %w", change.FQN(), err)
			}
			continue
		case *migrate.DropTable:
			err = m.DropTable(ctx, change.Schema, change.Name)
			if err != nil {
				return fmt.Errorf("apply changes: drop table %s: %w", change.FQN(), err)
			}
			continue
		case *migrate.RenameTable:
			b, err = m.renameTable(fmter, b, change)
		case *migrate.RenameColumn:
			b, err = m.renameColumn(fmter, b, change)
		case *migrate.DropConstraint:
			b, err = m.dropContraint(fmter, b, change)
		case *migrate.AddForeignKey:
			b, err = m.addForeignKey(fmter, b, change)
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
	fqn := rename.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}
	b = append(b, " RENAME TO "...)
	if b, err = bun.Ident(rename.NewName).AppendQuery(fmter, b); err != nil {
		return b, err
	}
	return b, nil
}

func (m *migrator) renameColumn(fmter schema.Formatter, b []byte, rename *migrate.RenameColumn) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	fqn := rename.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " RENAME COLUMN "...)
	if b, err = bun.Ident(rename.OldName).AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " TO "...)
	if b, err = bun.Ident(rename.NewName).AppendQuery(fmter, b); err != nil {
		return b, err
	}
	return b, nil
}

func (m *migrator) renameConstraint(fmter schema.Formatter, b []byte, rename *migrate.RenameConstraint) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	fqn := rename.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " RENAME CONSTRAINT "...)
	if b, err = bun.Ident(rename.OldName).AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " TO "...)
	if b, err = bun.Ident(rename.NewName).AppendQuery(fmter, b); err != nil {
		return b, err
	}
	return b, nil
}

func (m *migrator) dropContraint(fmter schema.Formatter, b []byte, drop *migrate.DropConstraint) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	fqn := drop.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " DROP CONSTRAINT "...)
	if b, err = bun.Ident(drop.ConstraintName).AppendQuery(fmter, b); err != nil {
		return b, err
	}
	return b, nil
}

func (m *migrator) addForeignKey(fmter schema.Formatter, b []byte, add *migrate.AddForeignKey) (_ []byte, err error) {
	b = append(b, "ALTER TABLE "...)
	fqn := add.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " ADD CONSTRAINT "...)
	if b, err = bun.Ident(add.ConstraintName).AppendQuery(fmter, b); err != nil {
		return b, err
	}

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
	fqn := colDef.FQN()
	if b, err = fqn.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	var i int
	appendAlterColumn := func() {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, " ALTER COLUMN "...)
		b, err = bun.Ident(colDef.Column).AppendQuery(fmter, b)
		i++
	}

	got, want := colDef.From, colDef.To

	if want.SQLType != got.SQLType {
		if appendAlterColumn(); err != nil {
			return b, err
		}
		b = append(b, " SET DATA TYPE "...)
		if b, err = want.AppendQuery(fmter, b); err != nil {
			return b, err
		}
	}

	if want.IsNullable != got.IsNullable {
		if appendAlterColumn(); err != nil {
			return b, err
		}
		if !want.IsNullable {
			b = append(b, " SET NOT NULL"...)
		} else {
			b = append(b, " DROP NOT NULL"...)
		}
	}

	if want.DefaultValue != got.DefaultValue {
		if appendAlterColumn(); err != nil {
			return b, err
		}
		if want.DefaultValue == "" {
			b = append(b, " DROP DEFAULT"...)
		} else {
			b = append(b, " SET DEFAULT "...)
			b = append(b, want.DefaultValue...)
		}
	}

	return b, nil
}
