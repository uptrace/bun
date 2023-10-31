package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"github.com/uptrace/bun/schema/inspector"
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
		var dialect inspector.Dialect
		dbDialect := db.Dialect()

		if id, ok := dbDialect.(inspector.Dialect); ok {
			dialect = id
		} else {
			t.Skipf("%q dialect does not implement inspector.Dialect", dbDialect.Name())
		}

		ctx := context.Background()
		mustResetModel(t, ctx, db, (*Book)(nil))

		dbInspector := dialect.Inspector(db)
		want := schema.State{
			Tables: []schema.TableDef{
				{
					Schema: "public",
					Name:   "books",
					Columns: map[string]schema.ColumnDef{
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

func getDatabaseInspectorOrSkip(tb testing.TB, db *bun.DB) schema.Inspector {
	dialect := db.Dialect()
	if id, ok := dialect.(inspector.Dialect); ok {
		return id.Inspector(db, migrationsTable, migrationLocksTable)
	}
	tb.Skipf("%q dialect does not implement inspector.Dialect", dialect.Name())
	return nil
}
