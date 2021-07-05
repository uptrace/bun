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
		ID  int64
		Str string
	}

	type User struct {
		ID   int64
		Name string
	}

	type Story struct {
		ID     int64
		Name   string
		UserID int64
		User   *User `bun:"rel:belongs-to"`
	}

	type SoftDelete struct {
		ID        int64
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
				ID     uint64
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
				WhereGroup(" OR ", func(q *bun.WhereQuery) {
					q.
						WhereOr("3").
						WhereOr("4").
						WhereGroup(" OR NOT ", func(q *bun.WhereQuery) {
							q.
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
			return db.NewDelete().Model(new(SoftDelete)).WherePK()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewDelete().Model(new(SoftDelete)).WherePK().ForceDelete()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete))
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete)).WhereDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			return db.NewSelect().Model(new(SoftDelete)).WhereAllWithDeleted()
		},
		func(db *bun.DB) schema.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewUpdate().
				Model(&models).
				Bulk()
		},
		func(db *bun.DB) schema.QueryAppender {
			type Model struct {
				ID   string `bun:",pk"`
				Name string
			}
			return db.NewCreateTable().
				Model((*Model)(nil)).
				IfNotExists()
		},
	}

	timeRE := regexp.MustCompile(`'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+(\+\d{2}:\d{2})?'`)

	testEachDB(t, func(t *testing.T, db *bun.DB) {
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
