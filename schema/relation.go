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
	// Base and Join can be explained with this query:
	//
	// SELECT * FROM base_table JOIN join_table

	Type      int
	Field     *Field
	JoinTable *Table
	BasePKs   []*Field
	JoinPKs   []*Field
	OnUpdate  string
	OnDelete  string
	Condition []string

	PolymorphicField *Field
	PolymorphicValue string

	M2MTable   *Table
	M2MBasePKs []*Field
	M2MJoinPKs []*Field
}

// References returns true if the table to which the Relation belongs needs to declare a foreign key constraint to create the relation.
// For other relations, the constraint is created in either the referencing table (1:N, 'has-many' relations) or a mapping table (N:N, 'm2m' relations).
func (r *Relation) References() bool {
	return r.Type == HasOneRelation || r.Type == BelongsToRelation
}

func (r *Relation) String() string {
	return fmt.Sprintf("relation=%s", r.Field.GoName)
}
