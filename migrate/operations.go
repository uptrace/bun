package migrate

import (
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
	FQN   schema.FQN
	Model interface{}
}

var _ Operation = (*CreateTable)(nil)

func (op *CreateTable) GetReverse() Operation {
	return &DropTable{FQN: op.FQN}
}

type DropTable struct {
	FQN schema.FQN
}

var _ Operation = (*DropTable)(nil)

func (op *DropTable) DependsOn(another Operation) bool {
	d, ok := another.(*DropConstraint)
	//
	return ok && ((d.FK.From.Schema == op.FQN.Schema && d.FK.From.Table == op.FQN.Table) ||
		(d.FK.To.Schema == op.FQN.Schema && d.FK.To.Table == op.FQN.Table))
}

// GetReverse for a DropTable returns a no-op migration. Logically, CreateTable is the reverse,
// but DropTable does not have the table's definition to create one.
//
// TODO: we can fetch table definitions for deleted tables
// from the database engine and execute them as a raw query.
func (op *DropTable) GetReverse() Operation {
	return &noop{}
}

type RenameTable struct {
	FQN     schema.FQN
	NewName string
}

var _ Operation = (*RenameTable)(nil)

func (op *RenameTable) GetReverse() Operation {
	return &RenameTable{
		FQN:     schema.FQN{Schema: op.FQN.Schema, Table: op.NewName},
		NewName: op.FQN.Table,
	}
}

// RenameColumn.
type RenameColumn struct {
	FQN     schema.FQN
	OldName string
	NewName string
}

var _ Operation = (*RenameColumn)(nil)

func (op *RenameColumn) GetReverse() Operation {
	return &RenameColumn{
		FQN:     op.FQN,
		OldName: op.NewName,
		NewName: op.OldName,
	}
}

func (op *RenameColumn) DependsOn(another Operation) bool {
	rt, ok := another.(*RenameTable)
	return ok && rt.FQN.Schema == op.FQN.Schema && rt.NewName == op.FQN.Table
}

type AddColumn struct {
	FQN    schema.FQN
	Column string
	ColDef sqlschema.Column
}

var _ Operation = (*AddColumn)(nil)

func (op *AddColumn) GetReverse() Operation {
	return &DropColumn{
		FQN:    op.FQN,
		Column: op.Column,
		ColDef: op.ColDef,
	}
}

type DropColumn struct {
	FQN    schema.FQN
	Column string
	ColDef sqlschema.Column
}

var _ Operation = (*DropColumn)(nil)

func (op *DropColumn) GetReverse() Operation {
	return &AddColumn{
		FQN:    op.FQN,
		Column: op.Column,
		ColDef: op.ColDef,
	}
}

func (op *DropColumn) DependsOn(another Operation) bool {
	// TODO: refactor
	if dc, ok := another.(*DropConstraint); ok {
		var fCol bool
		fCols := dc.FK.From.Column.Split()
		for _, c := range fCols {
			if c == op.Column {
				fCol = true
				break
			}
		}

		var tCol bool
		tCols := dc.FK.To.Column.Split()
		for _, c := range tCols {
			if c == op.Column {
				tCol = true
				break
			}
		}

		return (dc.FK.From.Schema == op.FQN.Schema && dc.FK.From.Table == op.FQN.Table && fCol) ||
			(dc.FK.To.Schema == op.FQN.Schema && dc.FK.To.Table == op.FQN.Table && tCol)
	}
	return false
}

// RenameConstraint.
type RenameConstraint struct {
	FK      sqlschema.FK
	OldName string
	NewName string
}

var _ Operation = (*RenameConstraint)(nil)

func (op *RenameConstraint) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *RenameConstraint) DependsOn(another Operation) bool {
	rt, ok := another.(*RenameTable)
	return ok && rt.FQN.Schema == op.FK.From.Schema && rt.NewName == op.FK.From.Table
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

func (op *AddForeignKey) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *AddForeignKey) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *RenameTable:
		// TODO: provide some sort of "DependsOn" method for FK
		return another.FQN.Schema == op.FK.From.Schema && another.NewName == op.FK.From.Table
	case *CreateTable:
		return (another.FQN.Schema == op.FK.To.Schema && another.FQN.Table == op.FK.To.Table) || // either it's the referencing one
			(another.FQN.Schema == op.FK.From.Schema && another.FQN.Table == op.FK.From.Table) // or the one being referenced
	}
	return false
}

func (op *AddForeignKey) GetReverse() Operation {
	return &DropConstraint{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

// TODO: Rename to DropForeignKey
// DropConstraint.
type DropConstraint struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*DropConstraint)(nil)

func (op *DropConstraint) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *DropConstraint) GetReverse() Operation {
	return &AddForeignKey{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

type AddUniqueConstraint struct {
	FQN    schema.FQN
	Unique sqlschema.Unique
}

var _ Operation = (*AddUniqueConstraint)(nil)

func (op *AddUniqueConstraint) GetReverse() Operation {
	return &DropUniqueConstraint{
		FQN:    op.FQN,
		Unique: op.Unique,
	}
}

func (op *AddUniqueConstraint) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *AddColumn:
		var sameColumn bool
		for _, column := range op.Unique.Columns.Split() {
			if column == another.Column {
				sameColumn = true
				break
			}
		}
		return op.FQN == another.FQN && sameColumn
	case *RenameTable:
		return op.FQN.Schema == another.FQN.Schema && op.FQN.Table == another.NewName
	case *DropUniqueConstraint:
		// We want to drop the constraint with the same name before adding this one.
		return op.FQN == another.FQN && op.Unique.Name == another.Unique.Name
	default:
		return false
	}

}

type DropUniqueConstraint struct {
	FQN    schema.FQN
	Unique sqlschema.Unique
}

var _ Operation = (*DropUniqueConstraint)(nil)

func (op *DropUniqueConstraint) DependsOn(another Operation) bool {
	if rename, ok := another.(*RenameTable); ok {
		return op.FQN.Schema == rename.FQN.Schema && op.FQN.Table == rename.NewName
	}
	return false
}

func (op *DropUniqueConstraint) GetReverse() Operation {
	return &AddUniqueConstraint{
		FQN:    op.FQN,
		Unique: op.Unique,
	}
}

// Change column type.
type ChangeColumnType struct {
	FQN    schema.FQN
	Column string
	From   sqlschema.Column
	To     sqlschema.Column
}

var _ Operation = (*ChangeColumnType)(nil)

func (op *ChangeColumnType) GetReverse() Operation {
	return &ChangeColumnType{
		FQN:    op.FQN,
		Column: op.Column,
		From:   op.To,
		To:     op.From,
	}
}

// noop is a migration that doesn't change the schema.
type noop struct{}

var _ Operation = (*noop)(nil)

func (*noop) GetReverse() Operation { return &noop{} }
