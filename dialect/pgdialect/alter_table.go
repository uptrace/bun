package pgdialect

import (
	"fmt"
	"strings"

	"github.com/uptrace/bun"
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

func (m *migrator) AppendSQL(b []byte, operation interface{}) (_ []byte, err error) {
	fmter := m.db.Formatter()

	// Append ALTER TABLE statement to the enclosed query bytes []byte.
	appendAlterTable := func(query []byte, fqn schema.FQN) []byte {
		query = append(query, "ALTER TABLE "...)
		query, _ = fqn.AppendQuery(fmter, query)
		return append(query, " "...)
	}

	switch change := operation.(type) {
	case *migrate.CreateTableOp:
		return m.AppendCreateTable(b, change.Model)
	case *migrate.DropTableOp:
		return m.AppendDropTable(b, change.FQN)
	case *migrate.RenameTableOp:
		b, err = m.renameTable(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.RenameColumnOp:
		b, err = m.renameColumn(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.AddColumnOp:
		b, err = m.addColumn(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.DropColumnOp:
		b, err = m.dropColumn(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.AddPrimaryKeyOp:
		b, err = m.addPrimaryKey(fmter, appendAlterTable(b, change.FQN), change.PrimaryKey)
	case *migrate.ChangePrimaryKeyOp:
		b, err = m.changePrimaryKey(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.DropPrimaryKeyOp:
		b, err = m.dropConstraint(fmter, appendAlterTable(b, change.FQN), change.PrimaryKey.Name)
	case *migrate.AddUniqueConstraintOp:
		b, err = m.addUnique(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.DropUniqueConstraintOp:
		b, err = m.dropConstraint(fmter, appendAlterTable(b, change.FQN), change.Unique.Name)
	case *migrate.ChangeColumnTypeOp:
		b, err = m.changeColumnType(fmter, appendAlterTable(b, change.FQN), change)
	case *migrate.AddForeignKeyOp:
		b, err = m.addForeignKey(fmter, appendAlterTable(b, change.FQN()), change)
	case *migrate.DropForeignKeyOp:
		b, err = m.dropConstraint(fmter, appendAlterTable(b, change.FQN()), change.ConstraintName)
	default:
		return nil, fmt.Errorf("append sql: unknown operation %T", change)
	}
	if err != nil {
		return nil, fmt.Errorf("append sql: %w", err)
	}
	return b, nil
}

func (m *migrator) renameTable(fmter schema.Formatter, b []byte, rename *migrate.RenameTableOp) (_ []byte, err error) {
	b = append(b, "RENAME TO "...)
	b = fmter.AppendName(b, rename.NewName)
	return b, nil
}

func (m *migrator) renameColumn(fmter schema.Formatter, b []byte, rename *migrate.RenameColumnOp) (_ []byte, err error) {
	b = append(b, "RENAME COLUMN "...)
	b = fmter.AppendName(b, rename.OldName)

	b = append(b, " TO "...)
	b = fmter.AppendName(b, rename.NewName)

	return b, nil
}

func (m *migrator) addColumn(fmter schema.Formatter, b []byte, add *migrate.AddColumnOp) (_ []byte, err error) {
	b = append(b, "ADD COLUMN "...)
	b = fmter.AppendName(b, add.Column)
	b = append(b, " "...)

	b, err = add.ColDef.AppendQuery(fmter, b)
	if err != nil {
		return nil, err
	}

	if add.ColDef.GetDefaultValue() != "" {
		b = append(b, " DEFAULT "...)
		b = append(b, add.ColDef.GetDefaultValue()...)
		b = append(b, " "...)
	}

	if add.ColDef.GetIsIdentity() {
		b = appendGeneratedAsIdentity(b)
	}

	return b, nil
}

func (m *migrator) dropColumn(fmter schema.Formatter, b []byte, drop *migrate.DropColumnOp) (_ []byte, err error) {
	b = append(b, "DROP COLUMN "...)
	b = fmter.AppendName(b, drop.Column)

	return b, nil
}

func (m *migrator) addPrimaryKey(fmter schema.Formatter, b []byte, pk sqlschema.PrimaryKey) (_ []byte, err error) {
	b = append(b, "ADD PRIMARY KEY ("...)
	b, _ = pk.Columns.AppendQuery(fmter, b)
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) changePrimaryKey(fmter schema.Formatter, b []byte, change *migrate.ChangePrimaryKeyOp) (_ []byte, err error) {
	b, _ = m.dropConstraint(fmter, b, change.Old.Name)
	b = append(b, ", "...)
	b, _ = m.addPrimaryKey(fmter, b, change.New)
	return b, nil
}

func (m *migrator) addUnique(fmter schema.Formatter, b []byte, change *migrate.AddUniqueConstraintOp) (_ []byte, err error) {
	b = append(b, "ADD CONSTRAINT "...)
	if change.Unique.Name != "" {
		b = fmter.AppendName(b, change.Unique.Name)
	} else {
		// Default naming scheme for unique constraints in Postgres is <table>_<column>_key
		b = fmter.AppendName(b, fmt.Sprintf("%s_%s_key", change.FQN.Table, change.Unique.Columns))
	}
	b = append(b, " UNIQUE ("...)
	b, _ = change.Unique.Columns.AppendQuery(fmter, b)
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) dropConstraint(fmter schema.Formatter, b []byte, name string) (_ []byte, err error) {
	b = append(b, "DROP CONSTRAINT "...)
	b = fmter.AppendName(b, name)

	return b, nil
}

func (m *migrator) addForeignKey(fmter schema.Formatter, b []byte, add *migrate.AddForeignKeyOp) (_ []byte, err error) {
	b = append(b, "ADD CONSTRAINT "...)

	name := add.ConstraintName
	if name == "" {
		colRef := add.ForeignKey.From
		columns := strings.Join(colRef.Column.Split(), "_")
		name = fmt.Sprintf("%s_%s_fkey", colRef.FQN.Table, columns)
	}
	b = fmter.AppendName(b, name)

	b = append(b, " FOREIGN KEY ("...)
	if b, err = add.ForeignKey.From.Column.AppendQuery(fmter, b); err != nil {
		return b, err
	}
	b = append(b, ")"...)

	b = append(b, " REFERENCES "...)
	if b, err = add.ForeignKey.To.FQN.AppendQuery(fmter, b); err != nil {
		return b, err
	}

	b = append(b, " ("...)
	if b, err = add.ForeignKey.To.Column.AppendQuery(fmter, b); err != nil {
		return b, err
	}
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) changeColumnType(fmter schema.Formatter, b []byte, colDef *migrate.ChangeColumnTypeOp) (_ []byte, err error) {
	// alterColumn never re-assigns err, so there is no need to check for err != nil after calling it
	var i int
	appendAlterColumn := func() {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, "ALTER COLUMN "...)
		b = fmter.AppendName(b, colDef.Column)
		i++
	}

	got, want := colDef.From, colDef.To

	inspector := m.db.Dialect().(sqlschema.InspectorDialect)
	if !inspector.EquivalentType(want, got) {
		appendAlterColumn()
		b = append(b, " SET DATA TYPE "...)
		if b, err = want.AppendQuery(fmter, b); err != nil {
			return b, err
		}
	}

	// Column must be declared NOT NULL before identity can be added.
	// Although PG can resolve the order of operations itself, we make this explicit in the query.
	if want.GetIsNullable() != got.GetIsNullable() {
		appendAlterColumn()
		if !want.GetIsNullable() {
			b = append(b, " SET NOT NULL"...)
		} else {
			b = append(b, " DROP NOT NULL"...)
		}
	}

	if want.GetIsIdentity() != got.GetIsIdentity() {
		appendAlterColumn()
		if !want.GetIsIdentity() {
			b = append(b, " DROP IDENTITY"...)
		} else {
			b = append(b, " ADD"...)
			b = appendGeneratedAsIdentity(b)
		}
	}

	if want.GetDefaultValue() != got.GetDefaultValue() {
		appendAlterColumn()
		if want.GetDefaultValue() == "" {
			b = append(b, " DROP DEFAULT"...)
		} else {
			b = append(b, " SET DEFAULT "...)
			b = append(b, want.GetDefaultValue()...)
		}
	}

	return b, nil
}
