package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
)

func TestDatabaseInspector_Inspect(t *testing.T) {

	type Book struct {
		bun.BaseModel `bun:"table:books"`

		ISBN   int    `bun:",pk,identity"`
		Author string `bun:",notnull,unique:title_author,default:'john doe'"`
		Title  string `bun:",notnull,unique:title_author"`
		Locale string `bun:",type:varchar(5),default:'en-GB'"`
		Pages  int8   `bun:"page_count,notnull,default:1"`
		Count  int32  `bun:"book_count,autoincrement"`
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		dbInspector, err := sqlschema.NewInspector(db)
		if err != nil {
			t.Skip(err)
		}

		ctx := context.Background()
		mustResetModel(t, ctx, db, (*Book)(nil))

		want := sqlschema.State{
			Tables: []sqlschema.Table{
				{
					Schema: "public",
					Name:   "books",
					Columns: map[string]sqlschema.Column{
						"isbn": {
							SQLType:         "bigint",
							IsPK:            true,
							IsNullable:      false,
							IsAutoIncrement: false,
							IsIdentity:      true,
							DefaultValue:    "",
						},
						"author": {
							SQLType:         "varchar",
							IsPK:            false,
							IsNullable:      false,
							IsAutoIncrement: false,
							IsIdentity:      false,
							DefaultValue:    "john doe",
						},
						"title": {
							SQLType:         "varchar",
							IsPK:            false,
							IsNullable:      false,
							IsAutoIncrement: false,
							IsIdentity:      false,
							DefaultValue:    "",
						},
						"locale": {
							SQLType:         "varchar(5)",
							IsPK:            false,
							IsNullable:      true,
							IsAutoIncrement: false,
							IsIdentity:      false,
							DefaultValue:    "en-GB",
						},
						"page_count": {
							SQLType:         "smallint",
							IsPK:            false,
							IsNullable:      false,
							IsAutoIncrement: false,
							IsIdentity:      false,
							DefaultValue:    "1",
						},
						"book_count": {
							SQLType:         "integer",
							IsPK:            false,
							IsNullable:      false,
							IsAutoIncrement: true,
							IsIdentity:      false,
							DefaultValue:    "",
						},
					},
				},
			},
		}

		got, err := dbInspector.Inspect(ctx)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}
