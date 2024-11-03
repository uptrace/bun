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
	d, ok := another.(*DropForeignKeyOp)
	//
	return ok && ((d.FK.From.Schema == op.FQN.Schema && d.FK.From.Table == op.FQN.Table) ||
		(d.FK.To.Schema == op.FQN.Schema && d.FK.To.Table == op.FQN.Table))
}

// GetReverse for a DropTable returns a no-op migration. Logically, CreateTable is the reverse,
// but DropTable does not have the table's definition to create one.
//
// TODO: we can fetch table definitions for deleted tables
// from the database engine and execute them as a raw query.
func (op *DropTableOp) GetReverse() Operation {
	return &noop{}
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
	rt, ok := another.(*RenameTableOp)
	return ok && rt.FQN.Schema == op.FQN.Schema && rt.NewName == op.FQN.Table
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
	// TODO: refactor
	switch drop := another.(type) {
	case *DropForeignKeyOp:
		var fCol bool
		fCols := drop.FK.From.Column.Split()
		for _, c := range fCols {
			if c == op.Column {
				fCol = true
				break
			}
		}

		var tCol bool
		tCols := drop.FK.To.Column.Split()
		for _, c := range tCols {
			if c == op.Column {
				tCol = true
				break
			}
		}

		return (drop.FK.From.Schema == op.FQN.Schema && drop.FK.From.Table == op.FQN.Table && fCol) ||
			(drop.FK.To.Schema == op.FQN.Schema && drop.FK.To.Table == op.FQN.Table && tCol)

	case *DropPrimaryKeyOp:
		return op.FQN == drop.FQN && drop.PK.Columns.Contains(op.Column)
	case *ChangePrimaryKeyOp:
		return op.FQN == drop.FQN && drop.Old.Columns.Contains(op.Column)
	}
	return false
}

// RenameForeignKeyOp.
type RenameForeignKeyOp struct {
	FK      sqlschema.FK
	OldName string
	NewName string
}

var _ Operation = (*RenameForeignKeyOp)(nil)

func (op *RenameForeignKeyOp) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *RenameForeignKeyOp) DependsOn(another Operation) bool {
	rt, ok := another.(*RenameTableOp)
	return ok && rt.FQN.Schema == op.FK.From.Schema && rt.NewName == op.FK.From.Table
}

func (op *RenameForeignKeyOp) GetReverse() Operation {
	return &RenameForeignKeyOp{
		FK:      op.FK,
		OldName: op.OldName,
		NewName: op.NewName,
	}
}

type AddForeignKeyOp struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*AddForeignKeyOp)(nil)

func (op *AddForeignKeyOp) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *AddForeignKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *RenameTableOp:
		// TODO: provide some sort of "DependsOn" method for FK
		return another.FQN.Schema == op.FK.From.Schema && another.NewName == op.FK.From.Table
	case *CreateTableOp:
		return (another.FQN.Schema == op.FK.To.Schema && another.FQN.Table == op.FK.To.Table) || // either it's the referencing one
			(another.FQN.Schema == op.FK.From.Schema && another.FQN.Table == op.FK.From.Table) // or the one being referenced
	}
	return false
}

func (op *AddForeignKeyOp) GetReverse() Operation {
	return &DropForeignKeyOp{
		FK:             op.FK,
		ConstraintName: op.ConstraintName,
	}
}

type DropForeignKeyOp struct {
	FK             sqlschema.FK
	ConstraintName string
}

var _ Operation = (*DropForeignKeyOp)(nil)

func (op *DropForeignKeyOp) FQN() schema.FQN {
	return schema.FQN{
		Schema: op.FK.From.Schema,
		Table:  op.FK.From.Table,
	}
}

func (op *DropForeignKeyOp) GetReverse() Operation {
	return &AddForeignKeyOp{
		FK:             op.FK,
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
		var sameColumn bool
		for _, column := range op.Unique.Columns.Split() {
			if column == another.Column {
				sameColumn = true
				break
			}
		}
		return op.FQN == another.FQN && sameColumn
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
	FQN schema.FQN
	PK  *sqlschema.PK
}

var _ Operation = (*DropPrimaryKeyOp)(nil)

func (op *DropPrimaryKeyOp) GetReverse() Operation {
	return &AddPrimaryKeyOp{
		FQN: op.FQN,
		PK:  op.PK,
	}
}

type AddPrimaryKeyOp struct {
	FQN schema.FQN
	PK  *sqlschema.PK
}

var _ Operation = (*AddPrimaryKeyOp)(nil)

func (op *AddPrimaryKeyOp) GetReverse() Operation {
	return &DropPrimaryKeyOp{
		FQN: op.FQN,
		PK:  op.PK,
	}
}

func (op *AddPrimaryKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *AddColumnOp:
		return op.FQN == another.FQN && op.PK.Columns.Contains(another.Column)
	}
	return false
}

type ChangePrimaryKeyOp struct {
	FQN schema.FQN
	Old *sqlschema.PK
	New *sqlschema.PK
}

var _ Operation = (*AddPrimaryKeyOp)(nil)

func (op *ChangePrimaryKeyOp) GetReverse() Operation {
	return &ChangePrimaryKeyOp{
		FQN: op.FQN,
		Old: op.New,
		New: op.Old,
	}
}

// noop is a migration that doesn't change the schema.
type noop struct{}

var _ Operation = (*noop)(nil)

func (*noop) GetReverse() Operation { return &noop{} }
