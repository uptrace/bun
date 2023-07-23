package dbtest_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/uptrace/bun"
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

	queries := []func(db *bun.DB) schema.QueryAppender{
		func(db *bun.DB) schema.QueryAppender {
			return db.NewValues(&Model{42, "hello"})
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewValues(&models)
		},

		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ModelTableExpr("?TableName AS ?TableAlias")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?PKs")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?TablePKs")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?Columns")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?TableColumns")
		},

		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Table("table")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().TableExpr("table")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).WhereOr("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Distinct().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().DistinctOn("foo").Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			query := db.NewSelect().Model(new(Model))
			return db.NewSelect().With("foo", query).Table("foo")
		},
		func(db *bun.DB) schema.QueryAppender {
			q1 := db.NewSelect().Model(new(Model)).Where("1")
			q2 := db.NewSelect().Model(new(Model))
			return q1.Union(q2)
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
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

		func(db *bun.DB) schema.QueryAppender {
			model := &Model{ID: 42, Str: "hello"}
			return db.NewInsert().Model(model)
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().Model(&models)
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []*Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().Model(&models)
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []*Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().Model(&models).On("CONFLICT DO NOTHING")
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			return db.
				NewInsert().
				Model(&map[string]interface{}{
					"id":  42,
					"str": "hello",
				}).
				Table("models")
		},
		func(db *bun.DB) schema.QueryAppender {
			src := db.NewValues(&[]map[string]interface{}{
				{"id": 42, "str": "hello"},
				{"id": 43, "str": "world"},
			})
			return db.NewInsert().With("src", src).TableExpr("dest").TableExpr("src")
		},

		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(Model)).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			model := &Model{ID: 42, Str: "hello"}
			return db.NewUpdate().Model(model).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().
				Model(&map[string]interface{}{"str": "hello"}).
				Table("models").
				Where("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
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

		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(Model)).WherePK()
		},

		func(db *bun.DB) schema.QueryAppender {
			return db.NewCreateTable().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID     uint64 `bun:",pk,autoincrement"`
				Struct struct{}
				Map    map[string]interface{}
				Slice  []string
				Array  []string `bun:",array"`
			}
			return db.NewCreateTable().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDropTable().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).ExcludeColumn("id")
		},
		func(db *bun.DB) schema.QueryAppender {
			type User struct {
				Name string `bun:",nullzero,notnull,default:\\'unknown\\'"`
			}
			return db.NewCreateTable().Model(new(User))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewCreateIndex().Unique().Index("title_idx").Table("films").Column("title")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewCreateIndex().
				Unique().
				Index("title_idx").
				Table("films").
				Column("title").
				Include("director", "rating")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Where("id IN (?)", bun.In([]int{1, 2, 3}))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Where("(id1, id2) IN (?)", bun.In([][]int{{1, 2}, {3, 4}}))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDropIndex().Concurrently().IfExists().Index("title_idx")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewAddColumn().Model(new(Model)).ColumnExpr("column_name VARCHAR(123)")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDropColumn().Model(new(Model)).Column("str")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewTruncateTable().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Story)).Relation("User")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model(new(Story)).
				Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
					q = q.ExcludeColumn("*")
					return q
				})
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model(new(Story)).
				Relation("User", func(q *bun.SelectQuery) *bun.SelectQuery {
					q = q.ExcludeColumn("id")
					return q
				})
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).WherePK().For("UPDATE")
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewUpdate().
				Model(&models).
				Table("_data").
				Where("model.id = _data.id")
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Int  int64     `bun:",nullzero"`
				Uint uint64    `bun:",nullzero"`
				Str  string    `bun:",nullzero"`
				Time time.Time `bun:",nullzero"`
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Int  int64     `bun:",nullzero,default:42"`
				Uint uint64    `bun:",nullzero,default:42"`
				Str  string    `bun:",nullzero,default:'hello'"`
				Time time.Time `bun:",nullzero,default:now()"`
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type mystr string
			type Model struct {
				Array []mystr `bun:",array"`
			}
			return db.NewInsert().Model(&Model{
				Array: []mystr{"foo", "bar"},
			})
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Ignore().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Replace().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []*Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().
				Model(&models).
				On("DUPLICATE KEY UPDATE").
				Set("str = upper(str)")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewCreateTable().
				Model(new(Model)).
				ForeignKey(`("profile_id") REFERENCES "profiles" ("id")`)
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Raw json.RawMessage
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Raw *json.RawMessage
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Bytes []byte
			}
			return db.NewInsert().Model(&Model{Bytes: make([]byte, 10)})
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Time bun.NullTime
			}
			models := make([]Model, 2)
			models[1].Time = bun.NullTime{Time: time.Unix(0, 0)}
			return db.NewValues(&models)
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Model(new(Model)).Value("foo", "?", "bar")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(Model)).Value("foo", "?", "bar").WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete1)).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete1)).WherePK().ForceDelete()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete1))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete1)).WhereDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete1)).WhereAllWithDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID   string `bun:",pk,autoincrement"`
				Name string
			}
			return db.NewCreateTable().
				Model((*Model)(nil)).
				IfNotExists()
		},
		func(db *bun.DB) schema.QueryAppender {
			type BaseModel struct {
				bun.BaseModel
				ID int64 `bun:",pk,autoincrement"`
			}
			type Model struct {
				BaseModel
			}
			return db.NewCreateTable().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				WhereGroup("", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where("a = 1").Where("b = 1")
				}).
				WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where("a = 2").Where("b = 2")
				})
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID int `bun:",pk"`
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			user := &User{Name: "Hello"}
			return db.NewUpdate().Model(user).Set("name = ?name").Where("id = ?id")
		},
		func(db *bun.DB) schema.QueryAppender {
			user := &User{ID: 42}
			return db.NewDelete().Model(user).Where("id = ?id")
		},
		func(db *bun.DB) schema.QueryAppender {
			user := &User{Name: "Hello"}
			return db.NewInsert().Model(user).On("CONFLICT DO UPDATE").Set("name = ?name")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.WhereOr("one").WhereOr("two")
			})
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "foo"},
			}
			return db.NewInsert().
				Model(&models).
				Value("str", "?", "custom")
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "foo"},
			}
			return db.NewUpdate().
				Model(&models).
				Value("str", "?", "custom").
				Bulk()
		},
		func(db *bun.DB) schema.QueryAppender {
			model := &Model{42, "hello"}
			return db.NewInsert().
				Model(model).
				On("CONFLICT (id) DO UPDATE")
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Foo   string `bun:",unique"`
				Bar   string `bun:",unique"`
				Hello string `bun:"unique:group"`
				World string `bun:"unique:group"`
			}
			return db.NewCreateTable().Model((*Model)(nil))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Time time.Time `bun:",notnull"`
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID int64
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Model(new(SoftDelete1)).On("CONFLICT DO NOTHING")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{ID: 42}).OmitZero().WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID   int64 `bun:",pk,autoincrement"`
				Time time.Time
			}
			return db.NewInsert().Model(&Model{ID: 123, Time: time.Unix(0, 0)})
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().ColumnExpr("id, name").Table("dest").Table("src")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete2)).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete2)).WherePK().ForceDelete()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete2))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete2)).WhereDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete2)).WhereAllWithDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Model(new(SoftDelete2)).On("CONFLICT DO NOTHING")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Model(&Model{}).Returning("")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewInsert().Model(&Model{Str: "hello"}).On("DUPLICATE KEY UPDATE")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewAddColumn().Model(new(Model)).
				ModelTableExpr("mytable").
				IfNotExists().
				ColumnExpr("column_name VARCHAR(123)")
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{ID: 1},
				{ID: 2},
			}
			return db.NewSelect().Model(&models).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{ID: 1, Str: "hello"},
				{ID: 2, Str: "world"},
			}
			return db.NewSelect().Model(&models).WherePK("id", "str")
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct{}

			return db.NewCreateTable().
				Model(&Model{}).
				ColumnExpr(`email VARCHAR`).
				ColumnExpr(`password VARCHAR`)
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewCreateTable().Model(new(Model)).PartitionBy("HASH (id)")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewCreateTable().Model(new(Model)).TableSpace("fasttablespace")
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			type ID string
			type Model struct {
				ID
			}
			return db.NewInsert().Model(&Model{ID: ID("embed")})
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Raw *json.RawMessage
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "foo"},
			}
			return db.NewInsert().
				Model(&models).
				Value("extra", "?", "custom")
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Time *time.Time
			}
			return db.NewInsert().Model(new(Model))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				Time *time.Time
			}
			tm := time.Unix(0, 0)
			return db.NewInsert().Model(&Model{Time: &tm})
		},
		func(db *bun.DB) schema.QueryAppender {
			values := [][]byte{
				[]byte("foo"),
				[]byte("bar"),
			}
			return db.NewSelect().Where("x IN (?)", bun.In(values))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				QueryBuilder().
				WhereGroup("", func(q bun.QueryBuilder) bun.QueryBuilder {
					return q.Where("a = 1").Where("b = 1")
				}).
				WhereGroup(" OR ", func(q bun.QueryBuilder) bun.QueryBuilder {
					return q.Where("a = 2").Where("b = 2")
				})
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).QueryBuilder().Where("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Model)).QueryBuilder().WhereOr("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete1)).QueryBuilder().WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete1)).QueryBuilder().WhereDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete1)).QueryBuilder().WhereAllWithDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(SoftDelete1)).QueryBuilder().WherePK().
				WhereGroup("", func(q bun.QueryBuilder) bun.QueryBuilder {
					return q.Where("a = 1").Where("b = 1")
				}).
				WhereGroup(" OR ", func(q bun.QueryBuilder) bun.QueryBuilder {
					return q.Where("a = 2").Where("b = 2")
				})
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(Model)).QueryBuilder().Where("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(Model)).QueryBuilder().WhereOr("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(SoftDelete1)).QueryBuilder().WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(SoftDelete1)).QueryBuilder().WherePK().WhereDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().
				Model(new(SoftDelete1)).
				QueryBuilder().
				WherePK().
				WhereAllWithDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(Model)).QueryBuilder().Where("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(Model)).QueryBuilder().WhereOr("id = 42")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete1)).QueryBuilder().WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete1)).QueryBuilder().WherePK().WhereDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().
				Model(new(SoftDelete1)).
				QueryBuilder().
				WherePK().
				WhereAllWithDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").UseIndex("ix1", "ix2")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Story)).Relation("User").UseIndexForJoin("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).Order("model.str ASC").UseIndexForOrderBy("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model(&Model{}).
				ColumnExpr("SUM(model.id) AS total_ids").
				Column("model.str").
				Group("model.str").
				UseIndexForGroupBy("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{
				ID:  1,
				Str: "hello",
			}).UseIndex("ix1", "ix2").Where("id = 3")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").IgnoreIndex("ix1", "ix2")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Story)).Relation("User").IgnoreIndexForJoin("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).Order("model.str ASC").IgnoreIndexForOrderBy("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model(&Model{}).
				ColumnExpr("SUM(model.id) AS total_ids").
				Column("model.str").
				Group("model.str").
				IgnoreIndexForGroupBy("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{
				ID:  1,
				Str: "hello",
			}).IgnoreIndex("ix1", "ix2").Where("id = 3")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").ForceIndex("ix1", "ix2")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(Story)).Relation("User").ForceIndexForJoin("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).Order("model.str ASC").ForceIndexForOrderBy("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model(&Model{}).
				ColumnExpr("SUM(model.id) AS total_ids").
				Column("model.str").Group("model.str").
				ForceIndexForGroupBy("ix1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{
				ID:  1,
				Str: "hello",
			}).ForceIndex("ix1", "ix2").Where("id = 3")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model((*Model)(nil)).
				Order("id DESC").
				Limit(20)
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model((*Model)(nil)).
				Order("id DESC").
				Offset(20).
				Limit(20)
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().
				Model((*Model)(nil)).
				Order("id DESC").
				Offset(20)
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(&Model{}).ColumnExpr("?PKs").
				UseIndex("ix1", "ix2").
				UseIndex("ix3")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{
				ID:  1,
				Str: "hello",
			}).UseIndex("ix1", "ix2").UseIndex("ix3").Where("id = 3")
		},
		func(db *bun.DB) schema.QueryAppender {
			type User struct {
				ID int64 `bun:",pk,autoincrement,identity"`
			}
			return db.NewCreateTable().Model(new(User))
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID           int64 `bun:",pk,autoincrement"`
				SoftDeleteID int64
				SoftDelete   *SoftDelete1 `bun:"rel:belongs-to"`
			}
			return db.NewSelect().Model(new(Model)).Relation("SoftDelete")
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID           int64 `bun:",pk,autoincrement"`
				SoftDeleteID int64
				SoftDelete   *SoftDelete2 `bun:"rel:belongs-to"`
			}
			return db.NewSelect().Model(new(Model)).Relation("SoftDelete")
		},
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
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
		func(db *bun.DB) schema.QueryAppender {
			// Non-positive VARCHAR length is illegal
			return db.NewCreateTable().Model((*Model)(nil)).Varchar(-20)
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().TableExpr("xxx").Set("foo = ?", bun.NullZero("")).Where("1")
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(new(Model)).OmitZero().WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{Str: ""}).OmitZero().WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{Str: ""}).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewUpdate().Model(&Model{42, ""}).OmitZero()
		},
	}

	timeRE := regexp.MustCompile(`'2\d{3}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?(\+\d{2}:\d{2})?'`)

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for i, fn := range queries {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				q := fn(db)

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
