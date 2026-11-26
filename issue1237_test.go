package bun

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/schema"
)

type issue1237TestDialect struct {
	schema.BaseDialect
	tables *schema.Tables
}

func newIssue1237TestDialect() *issue1237TestDialect {
	d := new(issue1237TestDialect)
	d.tables = schema.NewTables(d)
	return d
}

func (d *issue1237TestDialect) Init(*sql.DB) {}
func (d *issue1237TestDialect) Name() dialect.Name { return dialect.Invalid }
func (d *issue1237TestDialect) Features() feature.Feature { return feature.Returning }
func (d *issue1237TestDialect) Tables() *schema.Tables { return d.tables }
func (d *issue1237TestDialect) OnTable(*schema.Table) {}
func (d *issue1237TestDialect) IdentQuote() byte { return '"' }
func (d *issue1237TestDialect) AppendSequence(b []byte, _ *schema.Table, _ *schema.Field) []byte { return b }
func (d *issue1237TestDialect) DefaultVarcharLen() int { return 0 }
func (d *issue1237TestDialect) DefaultSchema() string { return "" }

type issue1237Wrapper struct {
	issue1237Order `bun:",extend"`
	Number         string
}

type issue1237Order struct {
	BaseModel `bun:"table:issue1237_orders"`
	Id        int             `bun:",pk,autoincrement"`
	Items     []issue1237Item `bun:"m2m:issue1237_order_to_items,join:Order=Item"`
}

type issue1237Item struct {
	BaseModel `bun:"table:issue1237_items"`
	Id        int    `bun:",pk,autoincrement"`
	Name      string
}

type issue1237OrderToItem struct {
	BaseModel `bun:"table:issue1237_order_to_items"`
	OrderId   int
	Order     *issue1237Order `bun:"rel:belongs-to,join:order_id=id"`
	ItemId    int
	Item      *issue1237Item  `bun:"rel:belongs-to,join:item_id=id"`
}

func TestM2MExtendPanic1237(t *testing.T) {
	db := NewDB(nil, newIssue1237TestDialect())
	db.RegisterModel((*issue1237OrderToItem)(nil))

	tbl := db.Table(reflect.TypeOf((*issue1237Wrapper)(nil)).Elem())
	require.NotNil(t, tbl)

	idField, ok := tbl.FieldMap["id"]
	require.True(t, ok)
	require.Equal(t, []int{0, 1}, idField.Index)

	rel, ok := tbl.Relations["Items"]
	require.True(t, ok)
	require.Equal(t, schema.ManyToManyRelation, rel.Type)
	require.Len(t, rel.BasePKs, 1)
	require.Equal(t, []int{0, 1}, rel.BasePKs[0].Index)

	t.Run("single model m2m query", func(t *testing.T) {
		saved := issue1237Wrapper{
			issue1237Order: issue1237Order{Id: 42},
			Number:         "inv-42",
		}
		q := db.NewSelect().
			Model(&saved).
			Relation("Items")

		joins := q.tableModel.getJoins()
		require.Len(t, joins, 1)
		require.Equal(t, schema.ManyToManyRelation, joins[0].Relation.Type)

		subq := joins[0].m2mQuery(db.NewSelect())
		require.NotNil(t, subq)
		queryBytes, err := subq.AppendQuery(db.gen, nil)
		require.NoError(t, err)
		queryStr := string(queryBytes)
		require.Contains(t, queryStr, `ON ("issue1237_order_to_item"."order_id") IN (42)`)
	})

	t.Run("slice model m2m query", func(t *testing.T) {
		saved := []issue1237Wrapper{
			{issue1237Order: issue1237Order{Id: 10}, Number: "inv-10"},
			{issue1237Order: issue1237Order{Id: 20}, Number: "inv-20"},
		}
		q := db.NewSelect().
			Model(&saved).
			Relation("Items")

		joins := q.tableModel.getJoins()
		require.Len(t, joins, 1)
		require.Equal(t, schema.ManyToManyRelation, joins[0].Relation.Type)

		subq := joins[0].m2mQuery(db.NewSelect())
		require.NotNil(t, subq)
		queryBytes, err := subq.AppendQuery(db.gen, nil)
		require.NoError(t, err)
		queryStr := string(queryBytes)
		require.Contains(t, queryStr, `ON ("issue1237_order_to_item"."order_id") IN (10, 20)`)
	})
}
