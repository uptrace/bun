package schema

import (
	"fmt"
)

const (
	InvalidRelation = iota
	HasOneRelation
	BelongsToRelation
	HasManyRelation
	ManyToManyRelation
)

type Relation struct {
	Type       int
	Field      *Field
	JoinTable  *Table
	BaseFields []*Field
	JoinFields []*Field
	OnUpdate   string
	OnDelete   string
	Condition  []string

	PolymorphicField *Field
	PolymorphicValue string

	M2MTable      *Table
	M2MBaseFields []*Field
	M2MJoinFields []*Field
}

// References returns true if the table to which the Relation belongs needs to declare a foreign key constraint to create the relation.
// For other relations, the constraint is created in either the referencing table (1:N, 'has-many' relations) or a mapping table (N:N, 'm2m' relations).
func (r *Relation) References() bool {
	return r.Type == HasOneRelation || r.Type == BelongsToRelation
}

func (r *Relation) String() string {
	return fmt.Sprintf("relation=%s", r.Field.GoName)
}
