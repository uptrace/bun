package alt

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

// Operation encapsulates the request to change a database definition
// and knowns which operation can revert it.
type Operation interface {
	GetReverse() Operation
}

// CreateTable
type CreateTable struct {
	Schema string
	Name   string
	Model  interface{}
}

var _ Operation = (*CreateTable)(nil)

func (op *CreateTable) GetReverse() Operation {
	return &DropTable{
		Schema: op.Schema,
		Name:   op.Name,
	}
}

type DropTable struct {
	Schema string
	Name   string
}

var _ Operation = (*DropTable)(nil)

// GetReverse for a DropTable returns a no-op migration. Logically, CreateTable is the reverse,
// but DropTable does not have the table's definition to create one.
//
// TODO: we can fetch table definitions for deleted tables
// from the database engine and execute them as a raw query.
func (op *DropTable) GetReverse() Operation {
	return &noop{}
}

type RenameTable struct {
	Schema  string
	OldName string
	NewName string
}

var _ Operation = (*RenameTable)(nil)
var _ sqlschema.Operation = (*RenameTable)(nil)

func (op *RenameTable) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.Schema,
		Table:  op.OldName,
	}
}

func (op *RenameTable) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return fmter.AppendQuery(b, "RENAME TO ?", bun.Ident(op.NewName)), nil
}

func (op *RenameTable) GetReverse() Operation {
	return &RenameTable{
		Schema:  op.Schema,
		OldName: op.NewName,
		NewName: op.OldName,
	}
}

// RenameColumn.
type RenameColumn struct {
	Schema  string
	Table   string
	OldName string
	NewName string
}

var _ Operation = (*RenameColumn)(nil)
var _ sqlschema.Operation = (*RenameColumn)(nil)

func (op *RenameColumn) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.Schema,
		Table:  op.Table,
	}
}

func (op *RenameColumn) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return fmter.AppendQuery(b, "RENAME COLUMN ? TO ?", bun.Ident(op.OldName), bun.Ident(op.NewName)), nil
}

func (op *RenameColumn) GetReverse() Operation {
	return &RenameColumn{
		Schema:  op.Schema,
		Table:   op.Table,
		OldName: op.NewName,
		NewName: op.OldName,
	}
}

func (op *RenameColumn) DependsOn(another Operation) bool {
	rt, ok := another.(*RenameTable)
	return ok && rt.Schema == op.Schema && rt.NewName == op.Table
}

// RenameConstraint
type RenameConstraint struct {
	FK      sqlschema.FK
	OldName string
	NewName string
}

var _ Operation = (*RenameConstraint)(nil)
var _ sqlschema.Operation = (*RenameConstraint)(nil)

func (op *RenameConstraint) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *RenameConstraint) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return fmter.AppendQuery(b, "RENAME CONSTRAINT ? TO ?", bun.Ident(op.OldName), bun.Ident(op.NewName)), nil
}

func (op *RenameConstraint) GetReverse() Operation {
	return &RenameConstraint{
		FK:      op.FK,
		OldName: op.OldName,
		NewName: op.NewName,
	}
}

type AddForeignKey struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*AddForeignKey)(nil)

func (op *AddForeignKey) GetReverse() Operation {
	return &DropForeignKey{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

type DropForeignKey struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*DropForeignKey)(nil)

func (op *DropForeignKey) GetReverse() Operation {
	return &AddForeignKey{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

type ChangeColumnType struct {
	Schema string
	Table  string
	Column string
	From   sqlschema.Column
	To     sqlschema.Column
}

var _ Operation = (*ChangeColumnType)(nil)

func (op *ChangeColumnType) GetReverse() Operation {
	return &ChangeColumnType{
		Schema: op.Schema,
		Table:  op.Table,
		Column: op.Column,
		From:   op.To,
		To:     op.From,
	}
}

// noop is a migration that doesn't change the schema.
type noop struct{}

var _ Operation = (*noop)(nil)

func (*noop) GetReverse() Operation { return &noop{} }
