package migrate

import (
	"fmt"

	"github.com/uptrace/bun/migrate/sqlschema"
)

// Operation encapsulates the request to change a database definition
// and knowns which operation can revert it.
//
// It is useful to define "monolith" Operations whenever possible,
// even though they a dialect may require several distinct steps to apply them.
// For example, changing a primary key involves first dropping the old constraint
// before generating the new one. Yet, this is only an implementation detail and
// passing a higher-level ChangePrimaryKeyOp will give the dialect more information
// about the applied change.
//
// Some operations might be irreversible due to technical limitations. Returning
// a *comment from GetReverse() will add an explanatory note to the generate migation file.
//
// To declare dependency on another Operation, operations should implement
// { DependsOn(Operation) bool } interface, which Changeset will use to resolve dependencies.
type Operation interface {
	GetReverse() Operation
}

// CreateTableOp creates a new table in the schema.
//
// It does not report dependency on any other migration and may be executed first.
// Make sure the dialect does not include FOREIGN KEY constraints in the CREATE TABLE
// statement, as those may potentially reference not-yet-existing columns/tables.
type CreateTableOp struct {
	FQN   sqlschema.FQN
	Model interface{}
}

var _ Operation = (*CreateTableOp)(nil)

func (op *CreateTableOp) GetReverse() Operation {
	return &DropTableOp{FQN: op.FQN}
}

// DropTableOp drops a database table. This operation is not reversible.
type DropTableOp struct {
	FQN sqlschema.FQN
}

var _ Operation = (*DropTableOp)(nil)

func (op *DropTableOp) DependsOn(another Operation) bool {
	drop, ok := another.(*DropForeignKeyOp)
	return ok && drop.ForeignKey.DependsOnTable(op.FQN)
}

// GetReverse for a DropTable returns a no-op migration. Logically, CreateTable is the reverse,
// but DropTable does not have the table's definition to create one.
func (op *DropTableOp) GetReverse() Operation {
	c := comment(fmt.Sprintf("WARNING: \"DROP TABLE %s\" cannot be reversed automatically because table definition is not available", op.FQN.String()))
	return &c
}

// RenameTableOp renames the table. Note, that changing the "schema" part of the table's FQN is not allowed.
type RenameTableOp struct {
	FQN     sqlschema.FQN
	NewName string
}

var _ Operation = (*RenameTableOp)(nil)

func (op *RenameTableOp) GetReverse() Operation {
	return &RenameTableOp{
		FQN:     sqlschema.FQN{Schema: op.FQN.Schema, Table: op.NewName},
		NewName: op.FQN.Table,
	}
}

// RenameColumnOp renames a column in the table. If the changeset includes a rename operation
// for the column's table, it should be executed first.
type RenameColumnOp struct {
	FQN     sqlschema.FQN
	OldName string
	NewName string
}

var _ Operation = (*RenameColumnOp)(nil)

func (op *RenameColumnOp) GetReverse() Operation {
	return &RenameColumnOp{
		FQN:     op.FQN,
		OldName: op.NewName,
		NewName: op.OldName,
	}
}

func (op *RenameColumnOp) DependsOn(another Operation) bool {
	rename, ok := another.(*RenameTableOp)
	return ok && op.FQN.Schema == rename.FQN.Schema && op.FQN.Table == rename.NewName
}

// AddColumnOp adds a new column to the table.
type AddColumnOp struct {
	FQN    sqlschema.FQN
	Column string
	ColDef sqlschema.Column
}

var _ Operation = (*AddColumnOp)(nil)

func (op *AddColumnOp) GetReverse() Operation {
	return &DropColumnOp{
		FQN:    op.FQN,
		Column: op.Column,
		ColDef: op.ColDef,
	}
}

// DropColumnOp drop a column from the table.
//
// While some dialects allow DROP CASCADE to drop dependent constraints,
// explicit handling on constraints is preferred for transparency and debugging.
// DropColumnOp depends on DropForeignKeyOp, DropPrimaryKeyOp, and ChangePrimaryKeyOp
// if any of the constraints is defined on this table.
type DropColumnOp struct {
	FQN    sqlschema.FQN
	Column string
	ColDef sqlschema.Column
}

var _ Operation = (*DropColumnOp)(nil)

func (op *DropColumnOp) GetReverse() Operation {
	return &AddColumnOp{
		FQN:    op.FQN,
		Column: op.Column,
		ColDef: op.ColDef,
	}
}

func (op *DropColumnOp) DependsOn(another Operation) bool {
	switch drop := another.(type) {
	case *DropForeignKeyOp:
		return drop.ForeignKey.DependsOnColumn(op.FQN, op.Column)
	case *DropPrimaryKeyOp:
		return op.FQN == drop.FQN && drop.PrimaryKey.Columns.Contains(op.Column)
	case *ChangePrimaryKeyOp:
		return op.FQN == drop.FQN && drop.Old.Columns.Contains(op.Column)
	}
	return false
}

// AddForeignKey adds a new FOREIGN KEY constraint.
type AddForeignKeyOp struct {
	ForeignKey     sqlschema.ForeignKey
	ConstraintName string
}

var _ Operation = (*AddForeignKeyOp)(nil)

func (op *AddForeignKeyOp) FQN() sqlschema.FQN {
	return op.ForeignKey.From.FQN
}

func (op *AddForeignKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *RenameTableOp:
		return op.ForeignKey.DependsOnTable(another.FQN) ||
			op.ForeignKey.DependsOnTable(sqlschema.FQN{Schema: another.FQN.Schema, Table: another.NewName})
	case *CreateTableOp:
		return op.ForeignKey.DependsOnTable(another.FQN)
	}
	return false
}

func (op *AddForeignKeyOp) GetReverse() Operation {
	return &DropForeignKeyOp{
		ForeignKey:     op.ForeignKey,
		ConstraintName: op.ConstraintName,
	}
}

// DropForeignKeyOp drops a FOREIGN KEY constraint.
type DropForeignKeyOp struct {
	ForeignKey     sqlschema.ForeignKey
	ConstraintName string
}

var _ Operation = (*DropForeignKeyOp)(nil)

func (op *DropForeignKeyOp) FQN() sqlschema.FQN {
	return op.ForeignKey.From.FQN
}

func (op *DropForeignKeyOp) GetReverse() Operation {
	return &AddForeignKeyOp{
		ForeignKey:     op.ForeignKey,
		ConstraintName: op.ConstraintName,
	}
}

// AddUniqueConstraintOp adds new UNIQUE constraint to the table.
type AddUniqueConstraintOp struct {
	FQN    sqlschema.FQN
	Unique sqlschema.Unique
}

var _ Operation = (*AddUniqueConstraintOp)(nil)

func (op *AddUniqueConstraintOp) GetReverse() Operation {
	return &DropUniqueConstraintOp{
		FQN:    op.FQN,
		Unique: op.Unique,
	}
}

func (op *AddUniqueConstraintOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *AddColumnOp:
		return op.FQN == another.FQN && op.Unique.Columns.Contains(another.Column)
	case *RenameTableOp:
		return op.FQN.Schema == another.FQN.Schema && op.FQN.Table == another.NewName
	case *DropUniqueConstraintOp:
		// We want to drop the constraint with the same name before adding this one.
		return op.FQN == another.FQN && op.Unique.Name == another.Unique.Name
	default:
		return false
	}

}

// DropUniqueConstraintOp drops a UNIQUE constraint.
type DropUniqueConstraintOp struct {
	FQN    sqlschema.FQN
	Unique sqlschema.Unique
}

var _ Operation = (*DropUniqueConstraintOp)(nil)

func (op *DropUniqueConstraintOp) DependsOn(another Operation) bool {
	if rename, ok := another.(*RenameTableOp); ok {
		return op.FQN.Schema == rename.FQN.Schema && op.FQN.Table == rename.NewName
	}
	return false
}

func (op *DropUniqueConstraintOp) GetReverse() Operation {
	return &AddUniqueConstraintOp{
		FQN:    op.FQN,
		Unique: op.Unique,
	}
}

// ChangeColumnTypeOp set a new data type for the column.
// The two types should be such that the data can be auto-casted from one to another.
// E.g. reducing VARCHAR lenght is not possible in most dialects.
// AutoMigrator does not enforce or validate these rules.
type ChangeColumnTypeOp struct {
	FQN    sqlschema.FQN
	Column string
	From   sqlschema.Column
	To     sqlschema.Column
}

var _ Operation = (*ChangeColumnTypeOp)(nil)

func (op *ChangeColumnTypeOp) GetReverse() Operation {
	return &ChangeColumnTypeOp{
		FQN:    op.FQN,
		Column: op.Column,
		From:   op.To,
		To:     op.From,
	}
}

// DropPrimaryKeyOp drops the table's PRIMARY KEY.
type DropPrimaryKeyOp struct {
	FQN        sqlschema.FQN
	PrimaryKey sqlschema.PrimaryKey
}

var _ Operation = (*DropPrimaryKeyOp)(nil)

func (op *DropPrimaryKeyOp) GetReverse() Operation {
	return &AddPrimaryKeyOp{
		FQN:        op.FQN,
		PrimaryKey: op.PrimaryKey,
	}
}

// AddPrimaryKeyOp adds a new PRIMARY KEY to the table.
type AddPrimaryKeyOp struct {
	FQN        sqlschema.FQN
	PrimaryKey sqlschema.PrimaryKey
}

var _ Operation = (*AddPrimaryKeyOp)(nil)

func (op *AddPrimaryKeyOp) GetReverse() Operation {
	return &DropPrimaryKeyOp{
		FQN:        op.FQN,
		PrimaryKey: op.PrimaryKey,
	}
}

func (op *AddPrimaryKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *AddColumnOp:
		return op.FQN == another.FQN && op.PrimaryKey.Columns.Contains(another.Column)
	}
	return false
}

// ChangePrimaryKeyOp changes the PRIMARY KEY of the table.
type ChangePrimaryKeyOp struct {
	FQN sqlschema.FQN
	Old sqlschema.PrimaryKey
	New sqlschema.PrimaryKey
}

var _ Operation = (*AddPrimaryKeyOp)(nil)

func (op *ChangePrimaryKeyOp) GetReverse() Operation {
	return &ChangePrimaryKeyOp{
		FQN: op.FQN,
		Old: op.New,
		New: op.Old,
	}
}

// comment denotes an Operation that cannot be executed.
//
// Operations, which cannot be reversed due to current technical limitations,
// may return &comment with a helpful message from their GetReverse() method.
//
// Chnagelog should skip it when applying operations or output as log message,
// and write it as an SQL comment when creating migration files.
type comment string

var _ Operation = (*comment)(nil)

func (c *comment) GetReverse() Operation { return c }
