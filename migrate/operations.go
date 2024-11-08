package migrate

import (
	"fmt"

	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

// Operation encapsulates the request to change a database definition
// and knowns which operation can revert it.
type Operation interface {
	GetReverse() Operation
}

// CreateTableOp
type CreateTableOp struct {
	FQN   schema.FQN
	Model interface{}
}

var _ Operation = (*CreateTableOp)(nil)

func (op *CreateTableOp) GetReverse() Operation {
	return &DropTableOp{FQN: op.FQN}
}

type DropTableOp struct {
	FQN schema.FQN
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

type RenameTableOp struct {
	FQN     schema.FQN
	NewName string
}

var _ Operation = (*RenameTableOp)(nil)

func (op *RenameTableOp) GetReverse() Operation {
	return &RenameTableOp{
		FQN:     schema.FQN{Schema: op.FQN.Schema, Table: op.NewName},
		NewName: op.FQN.Table,
	}
}

// RenameColumnOp.
type RenameColumnOp struct {
	FQN     schema.FQN
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

type AddColumnOp struct {
	FQN    schema.FQN
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

type DropColumnOp struct {
	FQN    schema.FQN
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

type AddForeignKeyOp struct {
	ForeignKey     sqlschema.ForeignKey
	ConstraintName string
}

var _ Operation = (*AddForeignKeyOp)(nil)

func (op *AddForeignKeyOp) FQN() schema.FQN {
	return op.ForeignKey.From.FQN
}

func (op *AddForeignKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *RenameTableOp:
		return op.ForeignKey.DependsOnTable(another.FQN) ||
			op.ForeignKey.DependsOnTable(schema.FQN{Schema: another.FQN.Schema, Table: another.NewName})
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

type DropForeignKeyOp struct {
	ForeignKey     sqlschema.ForeignKey
	ConstraintName string
}

var _ Operation = (*DropForeignKeyOp)(nil)

func (op *DropForeignKeyOp) FQN() schema.FQN {
	return op.ForeignKey.From.FQN
}

func (op *DropForeignKeyOp) GetReverse() Operation {
	return &AddForeignKeyOp{
		ForeignKey:     op.ForeignKey,
		ConstraintName: op.ConstraintName,
	}
}

type AddUniqueConstraintOp struct {
	FQN    schema.FQN
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

type DropUniqueConstraintOp struct {
	FQN    schema.FQN
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

// Change column type.
type ChangeColumnTypeOp struct {
	FQN    schema.FQN
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

type DropPrimaryKeyOp struct {
	FQN        schema.FQN
	PrimaryKey sqlschema.PrimaryKey
}

var _ Operation = (*DropPrimaryKeyOp)(nil)

func (op *DropPrimaryKeyOp) GetReverse() Operation {
	return &AddPrimaryKeyOp{
		FQN:        op.FQN,
		PrimaryKey: op.PrimaryKey,
	}
}

type AddPrimaryKeyOp struct {
	FQN        schema.FQN
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

type ChangePrimaryKeyOp struct {
	FQN schema.FQN
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
