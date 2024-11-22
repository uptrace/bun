package dbtest_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/internal/ordered"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

type Article struct {
	ISBN        int    `bun:",pk,identity"`
	Editor      string `bun:",notnull,unique:title_author,default:'john doe'"`
	Title       string `bun:",notnull,unique:title_author"`
	Locale      string `bun:",type:varchar(5),default:'en-GB'"`
	Pages       int8   `bun:"page_count,notnull,default:1"`
	Count       int32  `bun:"book_count,autoincrement"`
	PublisherID string `bun:"publisher_id,notnull"`
	AuthorID    int    `bun:"author_id,notnull"`

	// Publisher that published this article.
	Publisher *Publisher `bun:"rel:belongs-to,join:publisher_id=publisher_id"`

	// Author wrote this article.
	Author *Journalist `bun:"rel:belongs-to,join:author_id=author_id"`
}

type Office struct {
	bun.BaseModel `bun:"table:admin.offices"`
	Name          string `bun:"office_name,pk"`
	TennantID     string `bun:"publisher_id"`
	TennantName   string `bun:"publisher_name"`

	Tennant *Publisher `bun:"rel:has-one,join:publisher_id=publisher_id,join:publisher_name=publisher_name"`
}

type Publisher struct {
	ID        string    `bun:"publisher_id,pk,default:gen_random_uuid(),unique:office_fk"`
	Name      string    `bun:"publisher_name,notnull,unique:office_fk"`
	CreatedAt time.Time `bun:"created_at,default:current_timestamp"`

	// Writers write articles for this publisher.
	Writers []Journalist `bun:"m2m:publisher_to_journalists,join:Publisher=Author"`
}

// PublisherToJournalist describes which journalist work with each publisher.
// One publisher can also work with many journalists. It's an N:N (or m2m) relation.
type PublisherToJournalist struct {
	bun.BaseModel `bun:"table:publisher_to_journalists"`
	PublisherID   string `bun:"publisher_id,pk"`
	AuthorID      int    `bun:"author_id,pk"`

	Publisher *Publisher  `bun:"rel:belongs-to,join:publisher_id=publisher_id"`
	Author    *Journalist `bun:"rel:belongs-to,join:author_id=author_id"`
}

type Journalist struct {
	bun.BaseModel `bun:"table:authors"`
	ID            int    `bun:"author_id,pk,identity"`
	FirstName     string `bun:"first_name,notnull,unique:full_name"`
	LastName      string `bun:"last_name,notnull,unique:full_name"`
	Email         string `bun:"email,notnull,unique"`

	// Articles that this journalist has written.
	Articles []*Article `bun:"rel:has-many,join:author_id=author_id"`
}

func TestDatabaseInspector_Inspect(t *testing.T) {
	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		defaultSchema := db.Dialect().DefaultSchema()

		for _, tt := range []struct {
			name       string
			schemaName string
			wantTables *ordered.Map[string, sqlschema.Table]
			wantFKs    []sqlschema.ForeignKey
		}{
			{
				name:       "inspect default schema",
				schemaName: defaultSchema,
				// Tables come sorted alphabetically by schema and table.
				wantTables: ordered.NewMap[string, sqlschema.Table](
					// admin.offices should not be fetched, because it doesn't belong to the default schema.
					ordered.Pair[string, sqlschema.Table]{
						Key: "articles",
						Value: &sqlschema.BaseTable{
							Schema: defaultSchema,
							Name:   "articles",
							Columns: ordered.NewMap[string, sqlschema.Column](
								ordered.Pair[string, sqlschema.Column]{
									Key: "isbn",
									Value: &sqlschema.BaseColumn{
										SQLType:         "bigint",
										IsNullable:      false,
										IsAutoIncrement: false,
										IsIdentity:      true,
										DefaultValue:    "",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "editor",
									Value: &sqlschema.BaseColumn{
										SQLType:         sqltype.VarChar,
										IsNullable:      false,
										IsAutoIncrement: false,
										IsIdentity:      false,
										DefaultValue:    "john doe",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "title",
									Value: &sqlschema.BaseColumn{
										SQLType:         sqltype.VarChar,
										IsNullable:      false,
										IsAutoIncrement: false,
										IsIdentity:      false,
										DefaultValue:    "",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "locale",
									Value: &sqlschema.BaseColumn{
										SQLType:         sqltype.VarChar,
										VarcharLen:      5,
										IsNullable:      true,
										IsAutoIncrement: false,
										IsIdentity:      false,
										DefaultValue:    "en-GB",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "page_count",
									Value: &sqlschema.BaseColumn{
										SQLType:         "smallint",
										IsNullable:      false,
										IsAutoIncrement: false,
										IsIdentity:      false,
										DefaultValue:    "1",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "book_count",
									Value: &sqlschema.BaseColumn{
										SQLType:         "integer",
										IsNullable:      false,
										IsAutoIncrement: true,
										IsIdentity:      false,
										DefaultValue:    "",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "publisher_id",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "author_id",
									Value: &sqlschema.BaseColumn{
										SQLType: "bigint",
									},
								},
							),
							PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("isbn")},
							UniqueConstraints: []sqlschema.Unique{
								{Columns: sqlschema.NewColumns("editor", "title")},
							},
						},
					},
					ordered.Pair[string, sqlschema.Table]{
						Key: "authors",
						Value: &sqlschema.BaseTable{
							Schema: defaultSchema,
							Name:   "authors",
							Columns: ordered.NewMap[string, sqlschema.Column](
								ordered.Pair[string, sqlschema.Column]{
									Key: "author_id",
									Value: &sqlschema.BaseColumn{
										SQLType:    "bigint",
										IsIdentity: true,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "first_name",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "last_name",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "email",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
							),
							PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("author_id")},
							UniqueConstraints: []sqlschema.Unique{
								{Columns: sqlschema.NewColumns("first_name", "last_name")},
								{Columns: sqlschema.NewColumns("email")},
							},
						},
					},
					ordered.Pair[string, sqlschema.Table]{
						Key: "publisher_to_journalists",
						Value: &sqlschema.BaseTable{
							Schema: defaultSchema,
							Name:   "publisher_to_journalists",
							Columns: ordered.NewMap[string, sqlschema.Column](
								ordered.Pair[string, sqlschema.Column]{
									Key: "publisher_id",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "author_id",
									Value: &sqlschema.BaseColumn{
										SQLType: "bigint",
									},
								},
							),
							PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("publisher_id", "author_id")},
						},
					},
					ordered.Pair[string, sqlschema.Table]{
						Key: "publishers",
						Value: &sqlschema.BaseTable{
							Schema: defaultSchema,
							Name:   "publishers",
							Columns: ordered.NewMap[string, sqlschema.Column](
								ordered.Pair[string, sqlschema.Column]{
									Key: "publisher_id",
									Value: &sqlschema.BaseColumn{
										SQLType:      sqltype.VarChar,
										DefaultValue: "gen_random_uuid()",
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "publisher_name",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "created_at",
									Value: &sqlschema.BaseColumn{
										SQLType:      "timestamp",
										DefaultValue: "current_timestamp",
										IsNullable:   true,
									},
								},
							),
							PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("publisher_id")},
							UniqueConstraints: []sqlschema.Unique{
								{Columns: sqlschema.NewColumns("publisher_id", "publisher_name")},
							},
						},
					},
				),
				wantFKs: []sqlschema.ForeignKey{
					{
						From: sqlschema.NewColumnReference("articles", "publisher_id"),
						To:   sqlschema.NewColumnReference("publishers", "publisher_id"),
					},
					{
						From: sqlschema.NewColumnReference("articles", "author_id"),
						To:   sqlschema.NewColumnReference("authors", "author_id"),
					},
					{
						From: sqlschema.NewColumnReference("publisher_to_journalists", "publisher_id"),
						To:   sqlschema.NewColumnReference("publishers", "publisher_id"),
					},
					{
						From: sqlschema.NewColumnReference("publisher_to_journalists", "author_id"),
						To:   sqlschema.NewColumnReference("authors", "author_id"),
					},
				},
			},
			{
				name:       "inspect admin schema",
				schemaName: "admin",
				wantTables: ordered.NewMap[string, sqlschema.Table](
					ordered.Pair[string, sqlschema.Table]{
						Key: "offices",
						Value: &sqlschema.BaseTable{
							Schema: "admin",
							Name:   "offices",
							Columns: ordered.NewMap[string, sqlschema.Column](
								ordered.Pair[string, sqlschema.Column]{
									Key: "office_name",
									Value: &sqlschema.BaseColumn{
										SQLType: sqltype.VarChar,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "publisher_id",
									Value: &sqlschema.BaseColumn{
										SQLType:    sqltype.VarChar,
										IsNullable: true,
									},
								},
								ordered.Pair[string, sqlschema.Column]{
									Key: "publisher_name",
									Value: &sqlschema.BaseColumn{
										SQLType:    sqltype.VarChar,
										IsNullable: true,
									},
								},
							),
							PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("office_name")},
						},
					},
				),
				wantFKs: []sqlschema.ForeignKey{
					{
						From: sqlschema.NewColumnReference("offices", "publisher_name", "publisher_id"),
						To:   sqlschema.NewColumnReference("publishers", "publisher_name", "publisher_id"),
					},
				},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				db.RegisterModel((*PublisherToJournalist)(nil))

				dbInspector, err := sqlschema.NewInspector(db, sqlschema.WithSchemaName(tt.schemaName), sqlschema.WithExcludeTables(migrationsTable, migrationLocksTable))
				if err != nil {
					t.Skip(err)
				}

				ctx := context.Background()

				// Always create admin schema to test filtration is done correctly.
				mustCreateSchema(t, ctx, db, "admin")
				mustCreateTableWithFKs(t, ctx, db,
					// Order of creation matters:
					(*Journalist)(nil),            // does not reference other tables
					(*Publisher)(nil),             // does not reference other tables
					(*Office)(nil),                // references Publisher
					(*PublisherToJournalist)(nil), // references Journalist and Publisher
					(*Article)(nil),               // references Journalist and Publisher
				)

				got, err := dbInspector.Inspect(ctx)
				require.NoError(t, err)

				// State.FKs store their database names, which differ from dialect to dialect.
				// Because of that we compare FKs and Tables separately.
				gotTables := got.(sqlschema.BaseDatabase).Tables
				cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), tt.wantTables, gotTables)

				var fks []sqlschema.ForeignKey
				for fk := range got.GetForeignKeys() {
					fks = append(fks, fk)
				}
				require.ElementsMatch(t, tt.wantFKs, fks, "foreign keys differ: expected=listA, got=listB")
			})
		}
	})
}

func mustCreateTableWithFKs(tb testing.TB, ctx context.Context, db *bun.DB, models ...interface{}) {
	tb.Helper()
	for _, model := range models {
		create := db.NewCreateTable().Model(model).WithForeignKeys()
		_, err := create.Exec(ctx)
		require.NoError(tb, err, "arrange: must create table %q:", create.GetTableName())
		mustDropTableOnCleanup(tb, ctx, db, model)
	}
}

func mustCreateSchema(tb testing.TB, ctx context.Context, db *bun.DB, schema string) {
	tb.Helper()
	_, err := db.NewRaw("CREATE SCHEMA IF NOT EXISTS ?", bun.Ident(schema)).Exec(ctx)
	require.NoError(tb, err, "create schema %q:", schema)

	tb.Cleanup(func() {
		db.NewRaw("DROP SCHEMA IF EXISTS ?", bun.Ident(schema)).Exec(ctx)
	})
}

// cmpTables compares table schemas using dialect-specific equivalence checks for column types
// and reports the differences as t.Error().
func cmpTables(
	tb testing.TB,
	d sqlschema.InspectorDialect,
	want, got *ordered.Map[string, sqlschema.Table],
) {
	tb.Helper()

	require.ElementsMatch(tb, tableNames(want), tableNames(got), "different set of tables")

	// Now we are guaranteed to have the same tables.
	for _, tPair := range want.Pairs() {
		tableName, wantTable := tPair.Key, tPair.Value
		gotTable, ok := got.Load(tableName)
		require.True(tb, ok)
		cmpColumns(tb, d, wantTable.GetName(), wantTable.GetColumns(), gotTable.GetColumns())
		cmpConstraints(tb, wantTable.(*sqlschema.BaseTable), gotTable.(*sqlschema.BaseTable))
	}
}

// cmpColumns compares that column definitions on the tables are
func cmpColumns(
	tb testing.TB,
	d sqlschema.InspectorDialect,
	tableName string,
	want, got *ordered.Map[string, sqlschema.Column],
) {
	tb.Helper()
	var errs []string

	var missing []string
	for _, cPair := range want.Pairs() {
		colName := cPair.Key
		errorf := func(format string, args ...interface{}) {
			errs = append(errs, fmt.Sprintf("[%s.%s] "+format, append([]interface{}{tableName, colName}, args...)...))
		}
		wantCol := cPair.Value.(*sqlschema.BaseColumn)
		gotCol, ok := got.Value(colName).(*sqlschema.BaseColumn)
		if !ok {
			missing = append(missing, colName)
			continue
		}

		if !d.CompareType(wantCol, gotCol) {
			errorf("sql types are not equivalent:\n\t(+want)\t%s\n\t(-got)\t%s", formatType(wantCol), formatType(gotCol))
		}

		if wantCol.DefaultValue != gotCol.DefaultValue {
			errorf("default values differ:\n\t(+want)\t%s\n\t(-got)\t%s", wantCol.DefaultValue, gotCol.DefaultValue)
		}

		if wantCol.IsNullable != gotCol.IsNullable {
			errorf("isNullable:\n\t(+want)\t%t\n\t(-got)\t%t", wantCol.IsNullable, gotCol.IsNullable)
		}

		if wantCol.IsAutoIncrement != gotCol.IsAutoIncrement {
			errorf("IsAutoIncrement:\n\t(+want)\t%s\b\t(-got)\t%t", wantCol.IsAutoIncrement, gotCol.IsAutoIncrement)
		}

		if wantCol.IsIdentity != gotCol.IsIdentity {
			errorf("IsIdentity:\n\t(+want)\t%t\n\t(-got)\t%t", wantCol.IsIdentity, gotCol.IsIdentity)
		}
	}

	if len(missing) > 0 {
		errs = append(errs, fmt.Sprintf("%q has missing columns: %q", tableName, strings.Join(missing, "\", \"")))
	}

	var extra []string
	for _, colName := range got.Keys() {
		if _, ok := want.Load(colName); !ok {
			extra = append(extra, colName)
		}
	}

	if len(extra) > 0 {
		errs = append(errs, fmt.Sprintf("%q has extra columns: %q", tableName, strings.Join(extra, "\", \"")))
	}

	for _, errMsg := range errs {
		tb.Error(errMsg)
	}
}

// cmpConstraints compares constraints defined on the table with the expected ones.
func cmpConstraints(tb testing.TB, want, got *sqlschema.BaseTable) {
	tb.Helper()

	if want.PrimaryKey != nil {
		require.NotNilf(tb, got.PrimaryKey, "table %q missing primary key, want: (%s)", want.Name, want.PrimaryKey.Columns)
		require.Equalf(tb, want.PrimaryKey.Columns, got.PrimaryKey.Columns, "table %q has wrong primary key", want.Name)
	} else {
		require.Nilf(tb, got.PrimaryKey, "table %q shouldn't have a primary key", want.Name)
	}

	// Only keep columns included in each unique constraint for comparison.
	stripNames := func(uniques []sqlschema.Unique) (res []string) {
		for _, u := range uniques {
			res = append(res, u.Columns.String())
		}
		return
	}
	require.ElementsMatch(tb, stripNames(want.UniqueConstraints), stripNames(got.UniqueConstraints), "table %q does not have expected unique constraints (listA=want, listB=got)", want.Name)
}

func tableNames(tables *ordered.Map[string, sqlschema.Table]) []string {
	return tables.Keys()
}

func formatType(c sqlschema.Column) string {
	if c.GetVarcharLen() == 0 {
		return c.GetSQLType()
	}
	return fmt.Sprintf("%s(%d)", c.GetSQLType(), c.GetVarcharLen())
}

func TestBunModelInspector_Inspect(t *testing.T) {
	testEachDialect(t, func(t *testing.T, dialectName string, dialect schema.Dialect) {
		if _, ok := dialect.(sqlschema.InspectorDialect); !ok {
			t.Skip(dialectName + " is not sqlschema.InspectorDialect")
		}

		t.Run("default expressions are canonicalized", func(t *testing.T) {
			type Model struct {
				ID   string `bun:",notnull,default:RANDOM()"`
				Name string `bun:",notnull,default:'John Doe'"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName(dialect.DefaultSchema()))

			want := ordered.NewMap[string, sqlschema.Column](
				ordered.Pair[string, sqlschema.Column]{
					Key: "id",
					Value: &sqlschema.BaseColumn{
						SQLType:      sqltype.VarChar,
						DefaultValue: "random()",
					},
				},
				ordered.Pair[string, sqlschema.Column]{
					Key: "name",
					Value: &sqlschema.BaseColumn{
						SQLType:      sqltype.VarChar,
						DefaultValue: "John Doe",
					},
				},
			)

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			gotTables := got.GetTables()
			require.Equal(t, 1, gotTables.Len())
			for _, table := range gotTables.Values() {
				cmpColumns(t, dialect.(sqlschema.InspectorDialect), "model", want, table.GetColumns())
				return
			}
		})

		t.Run("parses custom varchar len", func(t *testing.T) {
			type Model struct {
				ID        string `bun:",notnull,type:text"`
				FirstName string `bun:",notnull,type:character varying(60)"`
				LastName  string `bun:",notnull,type:varchar(100)"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName(dialect.DefaultSchema()))

			want := ordered.NewMap[string, sqlschema.Column](
				ordered.Pair[string, sqlschema.Column]{
					Key: "id",
					Value: &sqlschema.BaseColumn{
						SQLType: "text",
					},
				},
				ordered.Pair[string, sqlschema.Column]{
					Key: "first_name",
					Value: &sqlschema.BaseColumn{
						SQLType:    "character varying",
						VarcharLen: 60,
					},
				},
				ordered.Pair[string, sqlschema.Column]{
					Key: "last_name",
					Value: &sqlschema.BaseColumn{
						SQLType:    "varchar",
						VarcharLen: 100,
					},
				},
			)

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			gotTables := got.GetTables()
			require.Equal(t, 1, gotTables.Len())
			for _, table := range gotTables.Values() {
				cmpColumns(t, dialect.(sqlschema.InspectorDialect), "model", want, table.GetColumns())
			}
		})

		t.Run("inspect unique constraints", func(t *testing.T) {
			type Model struct {
				ID        string `bun:",unique"`
				FirstName string `bun:"first_name,unique:full_name"`
				LastName  string `bun:"last_name,unique:full_name"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName(dialect.DefaultSchema()))

			want := &sqlschema.BaseTable{
				Name: "models",
				UniqueConstraints: []sqlschema.Unique{
					{Columns: sqlschema.NewColumns("id")},
					{Name: "full_name", Columns: sqlschema.NewColumns("first_name", "last_name")},
				},
			}

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			gotTables := got.GetTables()
			require.Equal(t, 1, gotTables.Len())
			for _, table := range gotTables.Values() {
				cmpConstraints(t, want, &table.(*sqlschema.BunTable).BaseTable)
				return
			}
		})
		t.Run("collects primary keys", func(t *testing.T) {
			type Model struct {
				ID       string    `bun:",pk"`
				Email    string    `bun:",pk"`
				Birthday time.Time `bun:",notnull"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName(dialect.DefaultSchema()))
			want := sqlschema.NewColumns("id", "email")

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			gotTables := got.GetTables()
			require.Equal(t, 1, gotTables.Len())
			for _, table := range gotTables.Values() {
				pk := table.GetPrimaryKey()
				require.NotNilf(t, pk, "did not register primary key, want (%s)", want)
				require.Equal(t, want, pk.Columns, "wrong primary key columns")
				return
			}
		})

		t.Run("separates schema and table name", func(t *testing.T) {
			type Model struct {
				bun.BaseModel `bun:"table:custom_schema.model"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName("custom_schema"))

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			gotTables := got.GetTables()
			require.Equal(t, 1, gotTables.Len())
			for _, table := range gotTables.Values() {
				require.Equal(t, "custom_schema", table.GetSchema(), "wrong schema name")
				require.Equal(t, "model", table.GetName(), "wrong table name")
				return
			}
		})

		t.Run("filters tables by schema", func(t *testing.T) {
			type KeepMe struct {
				bun.BaseModel `bun:"table:want.keep_me"`
			}

			type LoseMe struct {
				bun.BaseModel `bun:"table:lose_me"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*KeepMe)(nil), (*LoseMe)(nil))
			inspector := sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName("want"))

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			gotTables := got.GetTables()
			require.Equal(t, 1, gotTables.Len())
			for _, table := range gotTables.Values() {
				require.Equal(t, "want", table.GetSchema(), "wrong schema name")
				require.Equal(t, "keep_me", table.GetName(), "wrong table name")
				return
			}
		})
	})
}
