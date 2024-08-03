package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
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
	ID   string `bun:"publisher_id,pk,default:gen_random_uuid(),unique:office_fk"`
	Name string `bun:"publisher_name,unique,notnull,unique:office_fk"`

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
	FirstName     string `bun:",notnull"`
	LastName      string

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
						SQLType: "varchar",
						IsPK:    true,
					},
					"publisher_id": {
						SQLType:    "varchar",
						IsNullable: true,
					},
					"publisher_name": {
						SQLType:    "varchar",
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
					"publisher_id": {
						SQLType: "varchar",
					},
					"author_id": {
						SQLType: "bigint",
					},
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
						SQLType: "varchar",
					},
					"last_name": {
						SQLType:    "varchar",
						IsNullable: true,
					},
				},
			},
			{
				Schema: defaultSchema,
				Name:   "publisher_to_journalists",
				Columns: map[string]sqlschema.Column{
					"publisher_id": {
						SQLType: "varchar",
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
						SQLType:      "varchar",
						IsPK:         true,
						DefaultValue: "gen_random_uuid()",
					},
					"publisher_name": {
						SQLType: "varchar",
					},
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
		require.Equal(t, wantTables, got.Tables)

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
		require.NoError(tb, err, "must create table %q:", create.GetTableName())
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
