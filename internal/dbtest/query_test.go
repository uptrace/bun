package dbtest_test

import (
	"fmt"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/sqlfmt"
)

func TestQuery(t *testing.T) {
	type Model struct {
		ID  int `bun:",autoincrement"`
		Str string
	}

	queries := []func(db *bun.DB) sqlfmt.QueryAppender{
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewValues(&Model{42, "hello"})
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewValues(&models)
		},

		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ModelTableExpr("?TableName AS ?TableAlias")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?PKs")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?TablePKs")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?Columns")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model((*Model)(nil)).ColumnExpr("?TableColumns")
		},

		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Table("table")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().TableExpr("table")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model(new(Model)).WherePK()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model(new(Model)).Where("id = 42")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model(new(Model)).WhereOr("id = 42")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Distinct().Model(new(Model))
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().DistinctOn("foo").Model(new(Model))
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			query := db.NewSelect().Model(new(Model))
			return db.NewSelect().With("foo", query).Table("foo")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			q1 := db.NewSelect().Model(new(Model)).Where("1")
			q2 := db.NewSelect().Model(new(Model))
			return q1.Union(q2)
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
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
		func(db *bun.DB) sqlfmt.QueryAppender {
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

		func(db *bun.DB) sqlfmt.QueryAppender {
			model := &Model{ID: 42, Str: "hello"}
			return db.NewInsert().Model(model)
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().Model(&models)
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []*Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().Model(&models)
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []*Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().Model(&models).OnConflict("DO NOTHING")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []*Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewInsert().
				Model(&models).
				OnConflict("DO UPDATE").
				Set("model.str = EXCLUDED.str").
				Where("model.str IS NULL")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.
				NewInsert().
				Model(&map[string]interface{}{
					"id":  42,
					"str": "hello",
				}).
				Table("models")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			src := db.NewValues(&[]map[string]interface{}{
				{"id": 42, "str": "hello"},
				{"id": 43, "str": "world"},
			})
			return db.NewInsert().With("src", src).TableExpr("dest").TableExpr("src")
		},

		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewUpdate().Model(new(Model)).WherePK()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			model := &Model{ID: 42, Str: "hello"}
			return db.NewUpdate().Model(model).WherePK()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewUpdate().
				With("_data", db.NewValues(&models)).
				Model(&models).
				Set("model.str = _data.str").
				Where("model.id = _data.id")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.
				NewUpdate().
				Model(&map[string]interface{}{"str": "hello"}).
				Table("models").
				Where("id = 42")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
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

		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewDelete().Model(new(Model)).WherePK()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			model := &Model{ID: 42, Str: "hello"}
			return db.NewDelete().Model(model).WherePK()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewDelete().Model(&models).WherePK()
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			models := []Model{
				{42, "hello"},
				{43, "world"},
			}
			return db.NewDelete().Model(&models).WherePK().Where("name LIKE ?", "hello")
		},

		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewCreateTable().Model(new(Model))
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			type Model struct {
				ID     uint64 `bun:",autoincrement"`
				Struct struct{}
				Map    map[string]interface{}
				Slice  []string
				Array  []string `bun:",array"`
			}
			return db.NewCreateTable().Model(new(Model))
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewDropTable().Model(new(Model))
		},

		func(db *bun.DB) sqlfmt.QueryAppender {
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
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Model(new(Model)).ExcludeColumn("id")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			type User struct {
				Name string `bun:",nullzero,notnull,default:\\'unknown\\'"`
			}
			return db.NewCreateTable().Model(new(User))
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewCreateIndex().Unique().Index("title_idx").Table("films").Column("title")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewCreateIndex().
				Unique().
				Index("title_idx").
				Table("films").
				Column("title").
				Include("director", "rating")
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Where("id IN (?)", bun.In([]int{1, 2, 3}))
		},
		func(db *bun.DB) sqlfmt.QueryAppender {
			return db.NewSelect().Where("(id1, id2) IN (?)", bun.In([][]int{{1, 2}, {3, 4}}))
		},
	}

	for _, db := range dbs(t) {
		t.Run(db.Dialect().Name(), func(t *testing.T) {
			for i, fn := range queries {
				t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
					q := fn(db)

					query, err := q.AppendQuery(db.Formatter(), nil)
					require.NoError(t, err)
					cupaloy.SnapshotT(t, string(query))
				})
			}
		})
	}
}
