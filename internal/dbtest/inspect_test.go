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
		db.RegisterModel((*PublisherToJournalist)(nil))

		dbInspector, err := sqlschema.NewInspector(db)
		if err != nil {
			t.Skip(err)
		}

		ctx := context.Background()
		mustCreateSchema(t, ctx, db, "admin")
		mustCreateTableWithFKs(t, ctx, db,
			// Order of creation matters:
			(*Journalist)(nil),            // does not reference other tables
			(*Publisher)(nil),             // does not reference other tables
			(*Office)(nil),                // references Publisher
			(*PublisherToJournalist)(nil), // references Journalist and Publisher
			(*Article)(nil),               // references Journalist and Publisher
		)
		defaultSchema := db.Dialect().DefaultSchema()

		// Tables come sorted alphabetically by schema and table.
		wantTables := []sqlschema.Table{
			{
				Schema: "admin",
				Name:   "offices",
				Columns: map[string]sqlschema.Column{
					"office_name": {
						SQLType: sqltype.VarChar,
						IsPK:    true,
					},
					"publisher_id": {
						SQLType:    sqltype.VarChar,
						IsNullable: true,
					},
					"publisher_name": {
						SQLType:    sqltype.VarChar,
						IsNullable: true,
					},
				},
			},
			{
				Schema: defaultSchema,
				Name:   "articles",
				Columns: map[string]sqlschema.Column{
					"isbn": {
						SQLType:         "bigint",
						IsPK:            true,
						IsNullable:      false,
						IsAutoIncrement: false,
						IsIdentity:      true,
						DefaultValue:    "",
					},
					"editor": {
						SQLType:         sqltype.VarChar,
						IsPK:            false,
						IsNullable:      false,
						IsAutoIncrement: false,
						IsIdentity:      false,
						DefaultValue:    "john doe",
					},
					"title": {
						SQLType:         sqltype.VarChar,
						IsPK:            false,
						IsNullable:      false,
						IsAutoIncrement: false,
						IsIdentity:      false,
						DefaultValue:    "",
					},
					"locale": {
						SQLType:         sqltype.VarChar,
						VarcharLen:      5,
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
					"publisher_id": {
						SQLType: sqltype.VarChar,
					},
					"author_id": {
						SQLType: "bigint",
					},
				},
				UniqueContraints: []sqlschema.Unique{
					{Columns: sqlschema.NewComposite("editor", "title")},
				},
			},
			{
				Schema: defaultSchema,
				Name:   "authors",
				Columns: map[string]sqlschema.Column{
					"author_id": {
						SQLType:    "bigint",
						IsPK:       true,
						IsIdentity: true,
					},
					"first_name": {
						SQLType: sqltype.VarChar,
					},
					"last_name": {
						SQLType: sqltype.VarChar,
					},
					"email": {
						SQLType: sqltype.VarChar,
					},
				},
				UniqueContraints: []sqlschema.Unique{
					{Columns: sqlschema.NewComposite("first_name", "last_name")},
					{Columns: sqlschema.NewComposite("email")},
				},
			},
			{
				Schema: defaultSchema,
				Name:   "publisher_to_journalists",
				Columns: map[string]sqlschema.Column{
					"publisher_id": {
						SQLType: sqltype.VarChar,
						IsPK:    true,
					},
					"author_id": {
						SQLType: "bigint",
						IsPK:    true,
					},
				},
			},
			{
				Schema: defaultSchema,
				Name:   "publishers",
				Columns: map[string]sqlschema.Column{
					"publisher_id": {
						SQLType:      sqltype.VarChar,
						IsPK:         true,
						DefaultValue: "gen_random_uuid()",
					},
					"publisher_name": {
						SQLType: sqltype.VarChar,
					},
					"created_at": {
						SQLType:      "timestamp",
						DefaultValue: "current_timestamp",
						IsNullable:   true,
					},
				},
				UniqueContraints: []sqlschema.Unique{
					{Columns: sqlschema.NewComposite("publisher_id", "publisher_name")},
				},
			},
		}

		wantFKs := []sqlschema.FK{
			{ //
				From: sqlschema.C(defaultSchema, "articles", "publisher_id"),
				To:   sqlschema.C(defaultSchema, "publishers", "publisher_id"),
			},
			{
				From: sqlschema.C(defaultSchema, "articles", "author_id"),
				To:   sqlschema.C(defaultSchema, "authors", "author_id"),
			},
			{ //
				From: sqlschema.C("admin", "offices", "publisher_name", "publisher_id"),
				To:   sqlschema.C(defaultSchema, "publishers", "publisher_name", "publisher_id"),
			},
			{ //
				From: sqlschema.C(defaultSchema, "publisher_to_journalists", "publisher_id"),
				To:   sqlschema.C(defaultSchema, "publishers", "publisher_id"),
			},
			{ //
				From: sqlschema.C(defaultSchema, "publisher_to_journalists", "author_id"),
				To:   sqlschema.C(defaultSchema, "authors", "author_id"),
			},
		}

		got, err := dbInspector.Inspect(ctx)
		require.NoError(t, err)

		// State.FKs store their database names, which differ from dialect to dialect.
		// Because of that we compare FKs and Tables separately.
		cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, got.Tables)

		var fks []sqlschema.FK
		for fk := range got.FKs {
			fks = append(fks, fk)
		}
		require.ElementsMatch(t, wantFKs, fks)
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
func cmpTables(tb testing.TB, d sqlschema.InspectorDialect, want, got []sqlschema.Table) {
	tb.Helper()

	require.Equal(tb, tableNames(want), tableNames(got), "different set of tables")

	// Now we are guaranteed to have the same tables.
	for _, wt := range want {
		tableName := wt.Name
		// TODO(dyma): this will be simplified by map[string]Table
		var gt sqlschema.Table
		for i := range got {
			if got[i].Name == tableName {
				gt = got[i]
				break
			}
		}

		cmpColumns(tb, d, wt.Name, wt.Columns, gt.Columns)
		cmpConstraints(tb, wt, gt)
	}
}

// cmpColumns compares that column definitions on the tables are
func cmpColumns(tb testing.TB, d sqlschema.InspectorDialect, tableName string, want, got map[string]sqlschema.Column) {
	tb.Helper()
	var errs []string

	var missing []string
	for colName, wantCol := range want {
		errorf := func(format string, args ...interface{}) {
			errs = append(errs, fmt.Sprintf("[%s.%s] "+format, append([]interface{}{tableName, colName}, args...)...))
		}
		gotCol, ok := got[colName]
		if !ok {
			missing = append(missing, colName)
			continue
		}

		if !d.EquivalentType(wantCol, gotCol) {
			errorf("sql types are not equivalent:\n\t(+want)\t%s\n\t(-got)\t%s", formatType(wantCol), formatType(gotCol))
		}

		if wantCol.DefaultValue != gotCol.DefaultValue {
			errorf("default values differ:\n\t(+want)\t%s\n\t(-got)\t%s", wantCol.DefaultValue, gotCol.DefaultValue)
		}

		if wantCol.IsNullable != gotCol.IsNullable {
			errorf("isNullable:\n\t(+want)\t%s\n\t(-got)\t%s", wantCol.IsNullable, gotCol.IsNullable)
		}

		if wantCol.IsAutoIncrement != gotCol.IsAutoIncrement {
			errorf("IsAutoIncrement:\n\t(+want)\t%s\n\t(-got)\t%s", wantCol.IsAutoIncrement, gotCol.IsAutoIncrement)
		}

		if wantCol.IsIdentity != gotCol.IsIdentity {
			errorf("IsIdentity:\n\t(+want)\t%s\n\t(-got)\t%s", wantCol.IsIdentity, gotCol.IsIdentity)
		}
	}

	if len(missing) > 0 {
		errs = append(errs, fmt.Sprintf("%q has missing columns: %q", tableName, strings.Join(missing, "\", \"")))
	}

	var extra []string
	for colName := range got {
		if _, ok := want[colName]; !ok {
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
func cmpConstraints(tb testing.TB, want, got sqlschema.Table) {
	tb.Helper()

	// Only keep columns included in each unique constraint for comparison.
	stripNames := func(uniques []sqlschema.Unique) (res []string) {
		for _, u := range uniques {
			res = append(res, u.Columns.String())
		}
		return
	}
	require.ElementsMatch(tb, stripNames(want.UniqueContraints), stripNames(got.UniqueContraints), "table %q does not have expected unique constraints (listA=want, listB=got)", want.Name)
}

func tableNames(tables []sqlschema.Table) (names []string) {
	for i := range tables {
		names = append(names, tables[i].Name)
	}
	return
}

func formatType(c sqlschema.Column) string {
	if c.VarcharLen == 0 {
		return c.SQLType
	}
	return fmt.Sprintf("%s(%d)", c.SQLType, c.VarcharLen)
}

func TestSchemaInspector_Inspect(t *testing.T) {
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
			inspector := sqlschema.NewSchemaInspector(tables)

			want := map[string]sqlschema.Column{
				"id": {
					SQLType:      sqltype.VarChar,
					DefaultValue: "random()",
				},
				"name": {
					SQLType:      sqltype.VarChar,
					DefaultValue: "'John Doe'",
				},
			}

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			require.Len(t, got.Tables, 1)
			cmpColumns(t, dialect.(sqlschema.InspectorDialect), "model", want, got.Tables[0].Columns)
		})

		t.Run("parses custom varchar len", func(t *testing.T) {
			type Model struct {
				ID        string `bun:",notnull,type:text"`
				FirstName string `bun:",notnull,type:character varying(60)"`
				LastName  string `bun:",notnull,type:varchar(100)"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewSchemaInspector(tables)

			want := map[string]sqlschema.Column{
				"id": {
					SQLType: "text",
				},
				"first_name": {
					SQLType:    "character varying",
					VarcharLen: 60,
				},
				"last_name": {
					SQLType:    "varchar",
					VarcharLen: 100,
				},
			}

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			require.Len(t, got.Tables, 1)
			cmpColumns(t, dialect.(sqlschema.InspectorDialect), "model", want, got.Tables[0].Columns)
		})

		t.Run("inspect unique constraints", func(t *testing.T) {
			type Model struct {
				ID        string `bun:",unique"`
				FirstName string `bun:"first_name,unique:full_name"`
				LastName  string `bun:"last_name,unique:full_name"`
			}

			tables := schema.NewTables(dialect)
			tables.Register((*Model)(nil))
			inspector := sqlschema.NewSchemaInspector(tables)

			want := sqlschema.Table{
				Name: "models",
				UniqueContraints: []sqlschema.Unique{
					{Columns: sqlschema.NewComposite("id")},
					{Name: "full_name", Columns: sqlschema.NewComposite("first_name", "last_name")},
				},
			}

			got, err := inspector.Inspect(context.Background())
			require.NoError(t, err)

			require.Len(t, got.Tables, 1)
			cmpConstraints(t, want, got.Tables[0])
		})
	})
}
