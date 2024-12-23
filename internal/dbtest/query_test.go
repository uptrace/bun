package dbtest_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

func init() {
	snapshotsDir := filepath.Join("testdata", "snapshots")
	cupaloy.Global = cupaloy.Global.WithOptions(cupaloy.SnapshotSubdirectory(snapshotsDir))
}

func TestQuery(t *testing.T) {
	type Model struct {
		ID  int64 `bun:",pk,autoincrement"`
		Str string
	}

	type User struct {
		ID   int64 `bun:",pk,autoincrement"`
		Name string
	}

	type Story struct {
		ID     int64 `bun:",pk,autoincrement"`
		Name   string
		UserID int64
		User   *User `bun:"rel:belongs-to"`
	}

	type SoftDelete1 struct {
		bun.BaseModel `bun:"soft_deletes,alias:soft_delete"`

		ID        int64     `bun:",pk,autoincrement"`
		DeletedAt time.Time `bun:",soft_delete,nullzero"`
	}

	type SoftDelete2 struct {
		bun.BaseModel `bun:"soft_deletes,alias:soft_delete"`

		ID        int64     `bun:",pk,autoincrement"`
		DeletedAt time.Time `bun:",soft_delete"`
	}

	type test struct {
		id    int
		query func(db *bun.DB) schema.QueryAppender
	}

	tests := []test{
		{
			id: 0,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewValues(&Model{42, "hello"})
			},
		},
		{
			id: 1,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewValues(&models)
			},
		},
		{
			id: 2,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model((*Model)(nil)).ModelTableExpr("?TableName AS ?TableAlias")
			},
		},
		{
			id: 3,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?PKs")
			},
		},
		{
			id: 4,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?TablePKs")
			},
		},
		{
			id: 5,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?Columns")
			},
		},
		{
			id: 6,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?TableColumns")
			},
		},
		{
			id: 7,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect()
			},
		},
		{
			id: 8,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Table("table")
			},
		},
		{
			id: 9,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().TableExpr("table")
			},
		},
		{
			id: 10,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).WherePK()
			},
		},
		{
			id: 11,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).WherePK()
			},
		},
		{
			id: 12,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).WhereOr("id = 42")
			},
		},
		{
			id: 13,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Distinct().Model(new(Model))
			},
		},
		{
			id: 14,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().DistinctOn("foo").Model(new(Model))
			},
		},
		{
			id: 15,
			query: func(db *bun.DB) schema.QueryAppender {
				query := db.NewSelect().Model(new(Model))
				return db.NewSelect().With("foo", query).Table("foo")
			},
		},
		{
			id: 16,
			query: func(db *bun.DB) schema.QueryAppender {
				q1 := db.NewSelect().Model(new(Model)).Where("1")
				q2 := db.NewSelect().Model(new(Model))
				return q1.Union(q2)
			},
		},
		{
			id: 17,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewSelect().
					With("_data", db.NewValues(&models).WithOrder()).
					Model(&models).
					Where("model.id = _data.id").
					OrderExpr("_data._order")
			},
		},
		{
			id: 18,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewSelect().
					Model(&models).
					TableExpr("(?) AS (?Columns)", db.NewValues(&models).WithOrder()).
					Where("model.id = _data.id").
					OrderExpr("_data._order")
			},
		},
		{
			id: 19,
			query: func(db *bun.DB) schema.QueryAppender {
				model := &Model{ID: 42, Str: "hello"}
				return db.NewInsert().Model(model)
			},
		},
		{
			id: 20,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewInsert().Model(&models)
			},
		},
		{
			id: 21,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []*Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewInsert().Model(&models)
			},
		},
		{
			id: 22,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []*Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewInsert().Model(&models).On("CONFLICT DO NOTHING")
			},
		},
		{
			id: 23,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []*Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewInsert().
					Model(&models).
					On("CONFLICT DO UPDATE").
					Set("model.str = EXCLUDED.str").
					Where("model.str IS NULL")
			},
		},
		{
			id: 24,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.
					NewInsert().
					Model(&map[string]interface{}{
						"id":  42,
						"str": "hello",
					}).
					Table("models")
			},
		},
		{
			id: 25,
			query: func(db *bun.DB) schema.QueryAppender {
				src := db.NewValues(&[]map[string]interface{}{
					{"id": 42, "str": "hello"},
					{"id": 43, "str": "world"},
				})
				return db.NewInsert().With("src", src).TableExpr("dest").TableExpr("src")
			},
		},
		{
			id: 26,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(Model)).WherePK()
			},
		},
		{
			id: 27,
			query: func(db *bun.DB) schema.QueryAppender {
				model := &Model{ID: 42, Str: "hello"}
				return db.NewUpdate().Model(model).WherePK()
			},
		},
		{
			id: 28,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewUpdate().
					With("_data", db.NewValues(&models)).
					Model(&models).
					Table("_data").
					Set("model.str = _data.str").
					Where("model.id = _data.id")
			},
		},
		{
			id: 29,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().
					Model(&map[string]interface{}{"str": "hello"}).
					Table("models").
					Where("id = 42")
			},
		},
		{
			id: 30,
			query: func(db *bun.DB) schema.QueryAppender {
				src := db.NewValues(&[]map[string]interface{}{
					{"id": 42, "str": "hello"},
					{"id": 43, "str": "world"},
				})
				return db.NewUpdate().
					With("src", src).
					Table("dest", "src").
					Set("dest.str = src.str").
					Where("dest.id = src.id")
			},
		},
		{
			id: 31,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(Model)).WherePK()
			},
		},
		{
			id: 32,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateTable().Model(new(Model))
			},
		},
		{
			id: 33,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID     uint64 `bun:",pk,autoincrement"`
					Struct struct{}
					Map    map[string]interface{}
					Slice  []string
					Array  []string `bun:",array"`
				}
				return db.NewCreateTable().Model(new(Model))
			},
		},
		{
			id: 34,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDropTable().Model(new(Model))
			},
		},
		{
			id: 35,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).
					Where("1").
					WhereOr("2").
					WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.
							WhereOr("3").
							WhereOr("4").
							WhereGroup(" OR NOT ", func(q *bun.SelectQuery) *bun.SelectQuery {
								return q.
									WhereOr("5").
									WhereOr("6")
							})
					})
			},
		},
		{
			id: 36,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).ExcludeColumn("id")
			},
		},
		{
			id: 37,
			query: func(db *bun.DB) schema.QueryAppender {
				type User struct {
					Name string `bun:",nullzero,notnull,default:\\'unknown\\'"`
				}
				return db.NewCreateTable().Model(new(User))
			},
		},
		{
			id: 38,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateIndex().Unique().Index("title_idx").Table("films").Column("title")
			},
		},
		{
			id: 39,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateIndex().
					Unique().
					Index("title_idx").
					Table("films").
					Column("title").
					Include("director", "rating")
			},
		},
		{
			id: 40,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Where("id IN (?)", bun.In([]int{1, 2, 3}))
			},
		},
		{
			id: 41,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Where("(id1, id2) IN (?)", bun.In([][]int{{1, 2}, {3, 4}}))
			},
		},
		{
			id: 42,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDropIndex().Concurrently().IfExists().Index("title_idx")
			},
		},
		{
			id: 43,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewAddColumn().Model(new(Model)).ColumnExpr("column_name VARCHAR(123)")
			},
		},
		{
			id: 44,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDropColumn().Model(new(Model)).Column("str")
			},
		},
		{
			id: 45,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewTruncateTable().Model(new(Model))
			},
		},
		{
			id: 46,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Story)).Relation("User")
			},
		},
		{
			id: 47,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model(new(Story)).
					Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
						q = q.ExcludeColumn("*")
						return q
					})
			},
		},
		{
			id: 48,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model(new(Story)).
					Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
						q = q.ExcludeColumn("id")
						return q
					})
			},
		},
		{
			id: 49,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).WherePK().For("UPDATE")
			},
		},
		{
			id: 50,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewUpdate().
					Model(&models).
					Table("_data").
					Where("model.id = _data.id")
			},
		},
		{
			id: 51,
			query: func(db *bun.DB) schema.QueryAppender {
				// "nullzero" marshals zero values as DEFAULT or NULL (if DEFAULT placeholder is not supported)
				// DB drivers which support DEFAULT placeholder resolve it to NULL for columns that do not have a DEFAULT value.
				type Model struct {
					Int      int64     `bun:",nullzero"`
					Uint     uint64    `bun:",nullzero"`
					Str      string    `bun:",nullzero"`
					Time     time.Time `bun:",nullzero"`
					Bool     bool      `bun:",nullzero"`
					EmptyStr string    `bun:",nullzero"` // same as Str
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 52,
			query: func(db *bun.DB) schema.QueryAppender {
				// "nullzero,default" is equivalent to "default", marshalling zero values to DEFAULT
				type Model struct {
					Int      int64     `bun:",nullzero,default:42"`
					Uint     uint64    `bun:",nullzero,default:42"`
					Str      string    `bun:",nullzero,default:'hello'"`
					Time     time.Time `bun:",nullzero,default:now()"`
					Bool     bool      `bun:",nullzero,default:true"`
					EmptyStr string    `bun:",nullzero,default:''"`
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 53,
			query: func(db *bun.DB) schema.QueryAppender {
				type mystr string
				type Model struct {
					Array []mystr `bun:",array"`
				}
				return db.NewInsert().Model(&Model{
					Array: []mystr{"foo", "bar"},
				})
			},
		},
		{
			id: 54,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Ignore().Model(new(Model))
			},
		},
		{
			id: 55,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Replace().Model(new(Model))
			},
		},
		{
			id: 56,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []*Model{
					{42, "hello"},
					{43, "world"},
				}
				return db.NewInsert().
					Model(&models).
					On("DUPLICATE KEY UPDATE").
					Set("str = upper(str)")
			},
		},
		{
			id: 57,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateTable().
					Model(new(Model)).
					ForeignKey(`("profile_id") REFERENCES "profiles" ("id")`)
			},
		},
		{
			id: 58,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Raw json.RawMessage
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 59,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Raw *json.RawMessage
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 60,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Bytes []byte
				}
				return db.NewInsert().Model(&Model{Bytes: make([]byte, 10)})
			},
		},
		{
			id: 61,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Time bun.NullTime
				}
				models := make([]Model, 2)
				models[1].Time = bun.NullTime{Time: time.Unix(0, 0)}
				return db.NewValues(&models)
			},
		},
		{
			id: 62,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Model(new(Model)).Value("foo", "?", "bar")
			},
		},
		{
			id: 63,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(Model)).Value("foo", "?", "bar").WherePK()
			},
		},
		{
			id: 64,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(SoftDelete1)).WherePK()
			},
		},
		{
			id: 65,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(SoftDelete1)).WherePK().ForceDelete()
			},
		},
		{
			id: 66,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete1))
			},
		},
		{
			id: 67,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete1)).WhereDeleted()
			},
		},
		{
			id: 68,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete1)).WhereAllWithDeleted()
			},
		},
		{
			id: 69,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID   int64 `bun:",pk,autoincrement"`
					Str1 string
					Str2 string `bun:",skipupdate"`
				}
				models := []Model{
					{42, "hello", "skip"},
					{43, "world", "skip"},
				}
				return db.NewUpdate().
					Model(&models).
					Bulk()
			},
		},
		{
			id: 70,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID   string `bun:",pk,autoincrement"`
					Name string
				}
				return db.NewCreateTable().
					Model((*Model)(nil)).
					IfNotExists()
			},
		},
		{
			id: 71,
			query: func(db *bun.DB) schema.QueryAppender {
				type BaseModel struct {
					bun.BaseModel
					ID int64 `bun:",pk,autoincrement"`
				}
				type Model struct {
					BaseModel
				}
				return db.NewCreateTable().Model(new(Model))
			},
		},
		{
			id: 72,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					WhereGroup("", func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.Where("a = 1").Where("b = 1")
					}).
					WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.Where("a = 2").Where("b = 2")
					})
			},
		},
		{
			id: 73,
			query: func(db *bun.DB) schema.QueryAppender {
				params := struct {
					A     int
					B     float32
					Alias bun.Ident
				}{
					A:     1,
					B:     2.34,
					Alias: bun.Ident("sum"),
				}
				return db.NewSelect().Where("?a + ?b AS ?alias", params)
			},
		},
		{
			id: 74,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID int `bun:",pk"`
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 75,
			query: func(db *bun.DB) schema.QueryAppender {
				user := &User{Name: "Hello"}
				return db.NewUpdate().Model(user).Set("name = ?name").Where("id = ?id")
			},
		},
		{
			id: 76,
			query: func(db *bun.DB) schema.QueryAppender {
				user := &User{ID: 42}
				return db.NewDelete().Model(user).Where("id = ?id")
			},
		},
		{
			id: 77,
			query: func(db *bun.DB) schema.QueryAppender {
				user := &User{Name: "Hello"}
				return db.NewInsert().Model(user).On("CONFLICT DO UPDATE").Set("name = ?name")
			},
		},
		{
			id: 78,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.WhereOr("one").WhereOr("two")
				})
			},
		},
		{
			id: 79,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID   int64 `bun:",pk,autoincrement"`
					Str1 string
					Str2 string
				}

				models := []Model{
					{42, "hello", "world"},
					{43, "foo", "bar"},
				}
				return db.NewUpdate().
					Model(&models).
					Column("str2").
					Bulk()
			},
		},
		{
			id: 80,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "foo"},
				}
				return db.NewInsert().
					Model(&models).
					Value("str", "?", "custom")
			},
		},
		{
			id: 81,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "foo"},
				}
				return db.NewUpdate().
					Model(&models).
					Value("str", "?", "custom").
					Bulk()
			},
		},
		{
			id: 82,
			query: func(db *bun.DB) schema.QueryAppender {
				model := &Model{42, "hello"}
				return db.NewInsert().
					Model(model).
					On("CONFLICT (id) DO UPDATE")
			},
		},
		{
			id: 83,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Foo   string `bun:",unique"`
					Bar   string `bun:",unique"`
					Hello string `bun:"unique:group"`
					World string `bun:"unique:group"`
				}
				return db.NewCreateTable().Model((*Model)(nil))
			},
		},
		{
			id: 84,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Time time.Time `bun:",notnull"`
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 85,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID int64
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 86,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Model(new(SoftDelete1)).On("CONFLICT DO NOTHING")
			},
		},
		{
			id: 87,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{ID: 42}).OmitZero().WherePK()
			},
		},
		{
			id: 88,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID   int64 `bun:",pk,autoincrement"`
					Time time.Time
				}
				return db.NewInsert().Model(&Model{ID: 123, Time: time.Unix(0, 0)})
			},
		},
		{
			id: 89,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().ColumnExpr("id, name").Table("dest").Table("src")
			},
		},
		{
			id: 90,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(SoftDelete2)).WherePK()
			},
		},
		{
			id: 91,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(SoftDelete2)).WherePK().ForceDelete()
			},
		},
		{
			id: 92,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete2))
			},
		},
		{
			id: 93,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete2)).WhereDeleted()
			},
		},
		{
			id: 94,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete2)).WhereAllWithDeleted()
			},
		},
		{
			id: 95,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Model(new(SoftDelete2)).On("CONFLICT DO NOTHING")
			},
		},
		{
			id: 96,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Model(&Model{}).Returning("")
			},
		},
		{
			id: 97,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().Model(&Model{Str: "hello"}).On("DUPLICATE KEY UPDATE")
			},
		},
		{
			id: 98,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewAddColumn().Model(new(Model)).
					ModelTableExpr("mytable").
					IfNotExists().
					ColumnExpr("column_name VARCHAR(123)")
			},
		},
		{
			id: 99,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{ID: 1},
					{ID: 2},
				}
				return db.NewSelect().Model(&models).WherePK()
			},
		},
		{
			id: 100,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{ID: 1, Str: "hello"},
					{ID: 2, Str: "world"},
				}
				return db.NewSelect().Model(&models).WherePK("id", "str")
			},
		},
		{
			id: 101,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct{}

				return db.NewCreateTable().
					Model(&Model{}).
					ColumnExpr(`email VARCHAR`).
					ColumnExpr(`password VARCHAR`)
			},
		},
		{
			id: 102,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateTable().Model(new(Model)).PartitionBy("HASH (id)")
			},
		},
		{
			id: 103,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateTable().Model(new(Model)).TableSpace("fasttablespace")
			},
		},
		{
			id: 104,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID   int64     `bun:",pk,autoincrement"`
					Int  int64     `bun:",nullzero"`
					Uint uint64    `bun:",nullzero"`
					Str  string    `bun:",nullzero"`
					Time time.Time `bun:",nullzero"`
				}
				return db.NewUpdate().
					Model(new(Model)).
					Set("int = ?int").
					Set("uint = ?uint").
					Set("str = ?str").
					Set("time = ?time").
					WherePK()
			},
		},
		{
			id: 105,
			query: func(db *bun.DB) schema.QueryAppender {
				type ID string
				type Model struct {
					ID ID
				}
				return db.NewInsert().Model(&Model{ID: ID("embed")})
			},
		},
		{
			id: 106,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Raw *json.RawMessage
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 107,
			query: func(db *bun.DB) schema.QueryAppender {
				models := []Model{
					{42, "hello"},
					{43, "foo"},
				}
				return db.NewInsert().
					Model(&models).
					Value("extra", "?", "custom")
			},
		},
		{
			id: 108,
			query: func(db *bun.DB) schema.QueryAppender {
				type Item struct {
					Foo string
					Bar string
				}
				type Model struct {
					ID    int64
					Slice []Item `bun:",nullzero"`
				}
				return db.NewInsert().Model(&Model{ID: 123, Slice: make([]Item, 0)})
			},
		},
		{
			id: 109,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Time *time.Time
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 110,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					Time *time.Time
				}
				tm := time.Unix(0, 0)
				return db.NewInsert().Model(&Model{Time: &tm})
			},
		},
		{
			id: 111,
			query: func(db *bun.DB) schema.QueryAppender {
				values := [][]byte{
					[]byte("foo"),
					[]byte("bar"),
				}
				return db.NewSelect().Where("x IN (?)", bun.In(values))
			},
		},
		{
			id: 112,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					QueryBuilder().
					WhereGroup("", func(q bun.QueryBuilder) bun.QueryBuilder {
						return q.Where("a = 1").Where("b = 1")
					}).
					WhereGroup(" OR ", func(q bun.QueryBuilder) bun.QueryBuilder {
						return q.Where("a = 2").Where("b = 2")
					})
			},
		},
		{
			id: 113,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).QueryBuilder().Where("id = 42")
			},
		},
		{
			id: 114,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Model)).QueryBuilder().WhereOr("id = 42")
			},
		},
		{
			id: 115,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete1)).QueryBuilder().WherePK()
			},
		},
		{
			id: 116,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete1)).QueryBuilder().WhereDeleted()
			},
		},
		{
			id: 117,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(SoftDelete1)).QueryBuilder().WhereAllWithDeleted()
			},
		},
		{
			id: 118,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(SoftDelete1)).QueryBuilder().WherePK().
					WhereGroup("", func(q bun.QueryBuilder) bun.QueryBuilder {
						return q.Where("a = 1").Where("b = 1")
					}).
					WhereGroup(" OR ", func(q bun.QueryBuilder) bun.QueryBuilder {
						return q.Where("a = 2").Where("b = 2")
					})
			},
		},
		{
			id: 119,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(Model)).QueryBuilder().Where("id = 42")
			},
		},
		{
			id: 120,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(Model)).QueryBuilder().WhereOr("id = 42")
			},
		},
		{
			id: 121,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(SoftDelete1)).QueryBuilder().WherePK()
			},
		},
		{
			id: 122,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(SoftDelete1)).QueryBuilder().WherePK().WhereDeleted()
			},
		},
		{
			id: 123,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().
					Model(new(SoftDelete1)).
					QueryBuilder().
					WherePK().
					WhereAllWithDeleted()
			},
		},
		{
			id: 124,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().
					Model(new(SoftDelete1)).
					QueryBuilder().
					WherePK().
					WhereGroup("", func(q bun.QueryBuilder) bun.QueryBuilder {
						return q.Where("a = 1").Where("b = 1")
					}).
					WhereGroup(" OR ", func(q bun.QueryBuilder) bun.QueryBuilder {
						return q.Where("a = 2").Where("b = 2")
					})
			},
		},
		{
			id: 125,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(Model)).QueryBuilder().Where("id = 42")
			},
		},
		{
			id: 126,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(Model)).QueryBuilder().WhereOr("id = 42")
			},
		},
		{
			id: 127,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(SoftDelete1)).QueryBuilder().WherePK()
			},
		},
		{
			id: 128,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(new(SoftDelete1)).QueryBuilder().WherePK().WhereDeleted()
			},
		},
		{
			id: 129,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().
					Model(new(SoftDelete1)).
					QueryBuilder().
					WherePK().
					WhereAllWithDeleted()
			},
		},
		{
			id: 130,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID        int64 `bun:",pk"`
					UpdatedAt time.Time
				}
				return db.NewUpdate().
					Model(&Model{}).
					OmitZero().
					WherePK().
					Value("updated_at", "NOW()").
					Returning("*")
			},
		},
		{
			id: 131,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").UseIndex("ix1", "ix2")
			},
		},
		{
			id: 132,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Story)).Relation("User").UseIndexForJoin("ix1")
			},
		},
		{
			id: 133,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).Order("model.str ASC").UseIndexForOrderBy("ix1")
			},
		},
		{
			id: 134,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model(&Model{}).
					ColumnExpr("SUM(model.id) AS total_ids").
					Column("model.str").
					Group("model.str").
					UseIndexForGroupBy("ix1")
			},
		},
		{
			id: 135,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{
					ID:  1,
					Str: "hello",
				}).UseIndex("ix1", "ix2").Where("id = 3")
			},
		},
		{
			id: 136,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").IgnoreIndex("ix1", "ix2")
			},
		},
		{
			id: 137,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Story)).Relation("User").IgnoreIndexForJoin("ix1")
			},
		},
		{
			id: 138,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).Order("model.str ASC").IgnoreIndexForOrderBy("ix1")
			},
		},
		{
			id: 139,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model(&Model{}).
					ColumnExpr("SUM(model.id) AS total_ids").
					Column("model.str").
					Group("model.str").
					IgnoreIndexForGroupBy("ix1")
			},
		},
		{
			id: 140,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{
					ID:  1,
					Str: "hello",
				}).IgnoreIndex("ix1", "ix2").Where("id = 3")
			},
		},
		{
			id: 141,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").ForceIndex("ix1", "ix2")
			},
		},
		{
			id: 142,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(new(Story)).Relation("User").ForceIndexForJoin("ix1")
			},
		},
		{
			id: 143,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).Order("model.str ASC").ForceIndexForOrderBy("ix1")
			},
		},
		{
			id: 144,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model(&Model{}).
					ColumnExpr("SUM(model.id) AS total_ids").
					Column("model.str").Group("model.str").
					ForceIndexForGroupBy("ix1")
			},
		},
		{
			id: 145,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{
					ID:  1,
					Str: "hello",
				}).ForceIndex("ix1", "ix2").Where("id = 3")
			},
		},
		{
			id: 146,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model((*Model)(nil)).
					Order("id DESC").
					Limit(20)
			},
		},
		{
			id: 147,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model((*Model)(nil)).
					Order("id DESC").
					Offset(20).
					Limit(20)
			},
		},
		{
			id: 148,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model((*Model)(nil)).
					Order("id DESC").
					Offset(20)
			},
		},
		{
			id: 149,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").
					UseIndex("ix1", "ix2").
					UseIndex("ix3")
			},
		},
		{
			id: 150,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{
					ID:  1,
					Str: "hello",
				}).UseIndex("ix1", "ix2").UseIndex("ix3").Where("id = 3")
			},
		},
		{
			id: 151,
			query: func(db *bun.DB) schema.QueryAppender {
				type User struct {
					ID int64 `bun:",pk,autoincrement,identity"`
				}
				return db.NewCreateTable().Model(new(User))
			},
		},
		{
			id: 152,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID           int64 `bun:",pk,autoincrement"`
					SoftDeleteID int64
					SoftDelete   *SoftDelete1 `bun:"rel:belongs-to"`
				}
				return db.NewSelect().Model(new(Model)).Relation("SoftDelete")
			},
		},
		{
			id: 153,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID           int64 `bun:",pk,autoincrement"`
					SoftDeleteID int64
					SoftDelete   *SoftDelete2 `bun:"rel:belongs-to"`
				}
				return db.NewSelect().Model(new(Model)).Relation("SoftDelete")
			},
		},
		{
			id: 154,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID    int64 `bun:",pk,autoincrement"`
					Name  string
					Value string
				}

				newModels := []*Model{
					{Name: "A", Value: "world"},
					{Name: "B", Value: "test"},
				}

				return db.NewMerge().
					Model(new(Model)).
					With("_data", db.NewValues(&newModels)).
					Using("_data").
					On("?TableAlias.name = _data.name").
					WhenUpdate("MATCHED", func(q *bun.UpdateQuery) *bun.UpdateQuery {
						return q.Set("value = _data.value")
					}).
					WhenInsert("NOT MATCHED", func(q *bun.InsertQuery) *bun.InsertQuery {
						return q.Value("name", "_data.name").Value("value", "_data.value")
					}).
					Returning("$action")
			},
		},
		{
			id: 155,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID    int64 `bun:",pk,autoincrement"`
					Name  string
					Value string
				}

				newModels := []*Model{
					{Name: "A", Value: "world"},
					{Name: "B", Value: "test"},
				}

				return db.NewMerge().
					Model(new(Model)).
					With("_data", db.NewValues(&newModels)).
					Using("_data").
					On("?TableAlias.name = _data.name").
					WhenDelete("MATCHED").
					When("NOT MATCHED THEN INSERT (name, value) VALUES (_data.name, _data.value)").
					Returning("$action")
			},
		},
		{
			id: 156,
			query: func(db *bun.DB) schema.QueryAppender {
				// Note: not all dialects require specifying VARCHAR length
				type Model struct {
					// ID has the reflection-based type (DiscoveredSQLType) with default length
					ID string
					// Name has specific type and length defined (UserSQLType)
					Name string `bun:",type:varchar(50)"`
					// Title has user-defined type (UserSQLType) with default length
					Title string `bun:",type:varchar"`
				}
				// Set default VARCHAR length to 10
				return db.NewCreateTable().Model((*Model)(nil)).Varchar(10)
			},
		},
		{
			id: 157,
			query: func(db *bun.DB) schema.QueryAppender {
				// Non-positive VARCHAR length is illegal
				return db.NewCreateTable().Model((*Model)(nil)).Varchar(-20)
			},
		},
		{
			id: 158,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().TableExpr("xxx").Set("foo = ?", bun.NullZero("")).Where("1")
			},
		},
		{
			id: 159,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(new(Model)).OmitZero().WherePK()
			},
		},
		{
			id: 160,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{Str: ""}).OmitZero().WherePK()
			},
		},
		{
			id: 161,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{Str: ""}).WherePK()
			},
		},
		{
			id: 162,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&Model{42, ""}).OmitZero()
			},
		},
		{
			id: 163,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().
					Model((*Story)(nil)).
					Set("name = ?", "new-name").
					Join("JOIN user ON user.id = story.user_id").
					Where("user.id = ?", 1)
			},
		},
		{
			id: 164,
			query: func(db *bun.DB) schema.QueryAppender {
				q := db.NewCreateTable().Model(new(Story)).WithForeignKeys()

				// Check that building the query with .AppendQuery() multiple times does not add redundant FK constraints:
				// https://github.com/uptrace/bun/pull/941#discussion_r1443647857
				_ = q.String()
				return q
			},
		},
		{
			id: 165,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().Model(&struct {
					bun.BaseModel `bun:"table:accounts"`
					ID            int  `bun:"id,pk,autoincrement"`
					IsActive      bool `bun:"is_active,notnull,default:true"`
				}{
					ID:       1,
					IsActive: false,
				}).Column("is_active").WherePK()
			},
		},
		{
			id: 166,
			query: func(db *bun.DB) schema.QueryAppender {
				// "default" marshals zero values as DEFAULT or the specified default value
				type Model struct {
					Int      int64     `bun:",default:42"`
					Uint     uint64    `bun:",default:42"`
					Str      string    `bun:",default:'hello'"`
					Time     time.Time `bun:",default:now()"`
					Bool     bool      `bun:",default:true"`
					EmptyStr string    `bun:",default:''"`
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 167,
			query: func(db *bun.DB) schema.QueryAppender {
				// specified option names
				type Model struct {
					bun.BaseModel `bun:"table:table"`
					IsDefault     bool `bun:"column:default"`
				}
				return db.NewInsert().Model(new(Model))
			},
		},
		{
			id: 168,
			query: func(db *bun.DB) schema.QueryAppender {
				// DELETE ... ORDER BY ... (MySQL, MariaDB)
				return db.NewDelete().Model(new(Model)).WherePK().Order("id")
			},
		},
		{
			id: 169,
			query: func(db *bun.DB) schema.QueryAppender {
				// DELETE ... ORDER BY ... LIMIT ... (MySQL, MariaDB)
				return db.NewDelete().Model(new(Model)).WherePK().Order("id").Limit(1)
			},
		},
		{
			id: 170,
			query: func(db *bun.DB) schema.QueryAppender {
				// DELETE ... USING ... ORDER BY ... LIMIT ... (MySQL, MariaDB)
				return db.NewDelete().Model(new(Story)).TableExpr("archived_stories AS src").
					Where("src.id = story.id").Order("src.id").Limit(1)
			},
		},
		{
			id: 171,
			query: func(db *bun.DB) schema.QueryAppender {
				// UPDATE ... SET ... ORDER BY ... LIMIT ... (MySQL, MariaDB)
				return db.NewUpdate().Model(new(Story)).Set("name = ?", "new-name").WherePK().Order("id").Limit(1)
			},
		},
		{
			id: 172,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().Model(&Model{}).WherePK().Returning("*")
			},
		},
		{
			id: 173,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewAddColumn().
					Model(&Model{}).
					Comment("test").
					ColumnExpr("column_name VARCHAR(123)")
			},
		},
		{
			id: 174,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDropColumn().
					Model(&Model{}).
					Comment("test").
					Column("column_name")
			},
		},
		{
			id: 175,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDelete().
					Model(&Model{}).
					WherePK().
					Comment("test")
			},
		},
		{
			id: 176,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateIndex().
					Model(&Model{}).
					Unique().
					Index("index_name").
					Comment("test").
					Column("column_name")
			},
		},
		{
			id: 177,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDropIndex().
					Model(&Model{}).
					Comment("test").
					Index("index_name").
					Comment("test")
			},
		},
		{
			id: 178,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewInsert().
					Model(&Model{}).
					Comment("test").
					Value("column_name", "value")
			},
		},
		{
			id: 179,
			query: func(db *bun.DB) schema.QueryAppender {
				type Model struct {
					ID    int64 `bun:",pk,autoincrement"`
					Name  string
					Value string
				}

				newModels := []*Model{
					{Name: "A", Value: "world"},
					{Name: "B", Value: "test"},
				}

				return db.NewMerge().
					Model(new(Model)).
					With("_data", db.NewValues(&newModels)).
					Using("_data").
					On("?TableAlias.name = _data.name").
					WhenUpdate("MATCHED", func(q *bun.UpdateQuery) *bun.UpdateQuery {
						return q.Set("value = _data.value")
					}).
					WhenInsert("NOT MATCHED", func(q *bun.InsertQuery) *bun.InsertQuery {
						return q.Value("name", "_data.name").Value("value", "_data.value")
					}).
					Comment("test").
					Returning("$action")

			},
		},
		{
			id: 180,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewRaw("SELECT 1").Comment("test")
			},
		},
		{
			id: 181,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewSelect().
					Model(&Model{}).
					Comment("test")
			},
		},
		{
			id: 182,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewCreateTable().
					Model(&Model{}).
					Comment("test")
			},
		},
		{
			id: 183,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewDropTable().
					Model(&Model{}).
					Comment("test")
			},
		},
		{
			id: 184,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewTruncateTable().
					Model(&Model{}).
					Comment("test")
			},
		},
		{
			id: 185,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewUpdate().
					Model(&Model{}).
					Comment("test").
					Set("name = ?", "new-name").
					Where("id = ?", 1)
			},
		},
		{
			id: 186,
			query: func(db *bun.DB) schema.QueryAppender {
				return db.NewValues(&[]Model{{1, "hello"}}).
					Comment("test")
			},
		},
	}

	timeRE := regexp.MustCompile(`'2\d{3}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?(\+\d{2}:\d{2})?'`)

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%d", tt.id), func(t *testing.T) {
				q := tt.query(db)

				query, err := q.AppendQuery(db.Formatter(), nil)
				if err != nil {
					cupaloy.SnapshotT(t, err.Error())
				} else {
					query = timeRE.ReplaceAll(query, []byte("[TIME]"))
					cupaloy.SnapshotT(t, string(query))
				}
			})
		}
	})
}

func TestAlterTable(t *testing.T) {
	type Movie struct {
		bun.BaseModel `bun:"table:hobbies.movies"`
		ID            string
		Director      string `bun:"director,notnull"`
		Budget        int32
		ReleaseDate   time.Time
		HasOscar      bool
		Genre         string
	}

	schemaName := "hobbies"
	tableName := "movies"

	tests := []struct {
		name      string
		operation interface{}
	}{
		{name: "create table", operation: &migrate.CreateTableOp{
			TableName: tableName,
			Model:     (*Movie)(nil),
		}},
		{name: "drop table", operation: &migrate.DropTableOp{
			TableName: tableName,
		}},
		{name: "rename table", operation: &migrate.RenameTableOp{
			TableName: tableName,
			NewName:   "films",
		}},
		{name: "rename column", operation: &migrate.RenameColumnOp{
			TableName: tableName,
			OldName:   "has_oscar",
			NewName:   "has_awards",
		}},
		{name: "add column with default value", operation: &migrate.AddColumnOp{
			TableName:  tableName,
			ColumnName: "language",
			Column: &sqlschema.BaseColumn{
				SQLType:      "varchar",
				VarcharLen:   20,
				IsNullable:   false,
				DefaultValue: "'en-GB'",
			},
		}},
		{name: "add column with identity", operation: &migrate.AddColumnOp{
			TableName:  tableName,
			ColumnName: "n",
			Column: &sqlschema.BaseColumn{
				SQLType:    sqltype.BigInt,
				IsNullable: false,
				IsIdentity: true,
			},
		}},
		{name: "drop column", operation: &migrate.DropColumnOp{
			TableName:  tableName,
			ColumnName: "director",
			Column: &sqlschema.BaseColumn{
				SQLType:    sqltype.VarChar,
				IsNullable: false,
			},
		}},
		{name: "add unique constraint", operation: &migrate.AddUniqueConstraintOp{
			TableName: tableName,
			Unique: sqlschema.Unique{
				Name:    "one_genre_per_director",
				Columns: sqlschema.NewColumns("genre", "director"),
			},
		}},
		{name: "drop unique constraint", operation: &migrate.DropUniqueConstraintOp{
			TableName: tableName,
			Unique: sqlschema.Unique{
				Name:    "one_genre_per_director",
				Columns: sqlschema.NewColumns("genre", "director"),
			},
		}},
		{name: "change column type int to bigint", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "budget",
			From:      &sqlschema.BaseColumn{SQLType: sqltype.Integer},
			To:        &sqlschema.BaseColumn{SQLType: sqltype.BigInt},
		}},
		{name: "add default", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "budget",
			From:      &sqlschema.BaseColumn{DefaultValue: ""},
			To:        &sqlschema.BaseColumn{DefaultValue: "100"},
		}},
		{name: "drop default", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "budget",
			From:      &sqlschema.BaseColumn{DefaultValue: "100"},
			To:        &sqlschema.BaseColumn{DefaultValue: ""},
		}},
		{name: "make nullable", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "director",
			From:      &sqlschema.BaseColumn{IsNullable: false},
			To:        &sqlschema.BaseColumn{IsNullable: true},
		}},
		{name: "add notnull", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "budget",
			From:      &sqlschema.BaseColumn{IsNullable: true},
			To:        &sqlschema.BaseColumn{IsNullable: false},
		}},
		{name: "increase varchar length", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "language",
			From:      &sqlschema.BaseColumn{SQLType: "varchar", VarcharLen: 20},
			To:        &sqlschema.BaseColumn{SQLType: "varchar", VarcharLen: 255},
		}},
		{name: "add identity", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "id",
			From:      &sqlschema.BaseColumn{IsIdentity: false},
			To:        &sqlschema.BaseColumn{IsIdentity: true},
		}},
		{name: "drop identity", operation: &migrate.ChangeColumnTypeOp{
			TableName: tableName,
			Column:    "id",
			From:      &sqlschema.BaseColumn{IsIdentity: true},
			To:        &sqlschema.BaseColumn{IsIdentity: false},
		}},
		{name: "add primary key", operation: &migrate.AddPrimaryKeyOp{
			TableName: tableName,
			PrimaryKey: sqlschema.PrimaryKey{
				Name:    "new_pk",
				Columns: sqlschema.NewColumns("id"),
			},
		}},
		{name: "drop primary key", operation: &migrate.DropPrimaryKeyOp{
			TableName: tableName,
			PrimaryKey: sqlschema.PrimaryKey{
				Name:    "new_pk",
				Columns: sqlschema.NewColumns("id"),
			},
		}},
		{name: "change primary key", operation: &migrate.ChangePrimaryKeyOp{
			TableName: tableName,
			Old: sqlschema.PrimaryKey{
				Name:    "old_pk",
				Columns: sqlschema.NewColumns("id"),
			},
			New: sqlschema.PrimaryKey{
				Name:    "new_pk",
				Columns: sqlschema.NewColumns("director", "genre"),
			},
		}},
		{name: "add foreign key", operation: &migrate.AddForeignKeyOp{
			ConstraintName: "genre_description",
			ForeignKey: sqlschema.ForeignKey{
				From: sqlschema.NewColumnReference("movies", "genre"),
				To:   sqlschema.NewColumnReference("film_genres", "id"),
			},
		}},
		{name: "drop foreign key", operation: &migrate.DropForeignKeyOp{
			ConstraintName: "genre_description",
			ForeignKey: sqlschema.ForeignKey{
				From: sqlschema.NewColumnReference("movies", "genre"),
				To:   sqlschema.NewColumnReference("film_genres", "id"),
			},
		}},
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		migrator, err := sqlschema.NewMigrator(db, schemaName)
		if err != nil {
			t.Skip(err)
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b := internal.MakeQueryBytes()

				b, err := migrator.AppendSQL(b, tt.operation)
				require.NoError(t, err, "append sql")

				if err == nil {
					cupaloy.SnapshotT(t, string(b))
				} else {
					cupaloy.SnapshotT(t, err.Error())
				}
			})
		}
	})
}
