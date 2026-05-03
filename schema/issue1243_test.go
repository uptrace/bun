package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Models for TestNestedRelationWithSharedComposition (issue #1243).
// These must be at package level to allow circular type references
// between Business and Lead via shared embedded relation structs.

type idField struct {
	ID int64 `bun:",pk,autoincrement"`
}

type business struct {
	idField
	leadRelation
}

type businessRelation struct {
	BusinessID int64     `bun:",nullzero"`
	Business   *business `bun:"rel:belongs-to,join:business_id=id"`
}

type lead struct {
	idField
	businessRelation
}

type leadRelation struct {
	LeadID int64 `bun:",nullzero"`
	Lead   *lead `bun:"rel:belongs-to,join:lead_id=id"`
}

type agent struct {
	idField
	businessRelation

	Associations []*link `bun:"rel:has-many,join:id=agent_id"`
}

type agentRelation struct {
	AgentID int64  `bun:",nullzero"`
	Agent   *agent `bun:"rel:belongs-to,join:agent_id=id"`
}

type link struct {
	idField
	agentRelation
	leadRelation
}

func TestNestedRelationWithSharedComposition(t *testing.T) {
	dialect := newNopDialect()
	tables := NewTables(dialect)

	// Initialize Link table, which triggers circular initialization:
	// Link -> AgentRelation -> Agent -> BusinessRelation -> Business
	//   -> LeadRelation -> Lead -> BusinessRelation -> Business (cycle)
	linkTable := tables.Get(reflect.TypeFor[*link]())

	require.Contains(t, linkTable.StructMap, "lead",
		"Link should have 'lead' in StructMap from embedded LeadRelation")

	leadTable := linkTable.StructMap["lead"].Table
	require.Contains(t, leadTable.StructMap, "business",
		"Lead should have 'business' in StructMap from embedded BusinessRelation")

	// This is the lookup that fails in issue #1243:
	// when scanning column "lead__business__id" from a nested Relation() query.
	field := linkTable.LookupField("lead__business__id")
	require.NotNil(t, field,
		"LookupField('lead__business__id') must resolve through Lead -> Business")
}
