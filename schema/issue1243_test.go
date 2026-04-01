package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Models for TestNestedRelationWithSharedComposition (issue #1243).
// These must be at package level to allow circular type references
// between Business and Lead via shared embedded relation structs.

type issue1243IDField struct {
	ID int64 `bun:",pk,autoincrement"`
}

type issue1243Business struct {
	issue1243IDField
	issue1243LeadRelation
}

type issue1243BusinessRelation struct {
	BusinessID int64              `bun:",nullzero"`
	Business   *issue1243Business `bun:"rel:belongs-to,join:business_id=id"`
}

type issue1243Lead struct {
	issue1243IDField
	issue1243BusinessRelation
}

type issue1243LeadRelation struct {
	LeadID int64          `bun:",nullzero"`
	Lead   *issue1243Lead `bun:"rel:belongs-to,join:lead_id=id"`
}

type issue1243Agent struct {
	issue1243IDField
	issue1243BusinessRelation

	Associations []*issue1243Link `bun:"rel:has-many,join:id=agent_id"`
}

type issue1243AgentRelation struct {
	AgentID int64           `bun:",nullzero"`
	Agent   *issue1243Agent `bun:"rel:belongs-to,join:agent_id=id"`
}

type issue1243Link struct {
	issue1243IDField
	issue1243AgentRelation
	issue1243LeadRelation
}

func TestNestedRelationWithSharedComposition(t *testing.T) {
	dialect := newNopDialect()
	tables := NewTables(dialect)

	// Initialize Link table, which triggers circular initialization:
	// Link -> AgentRelation -> Agent -> BusinessRelation -> Business
	//   -> LeadRelation -> Lead -> BusinessRelation -> Business (cycle)
	linkTable := tables.Get(reflect.TypeFor[*issue1243Link]())

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
