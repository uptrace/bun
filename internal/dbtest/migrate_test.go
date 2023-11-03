package dbtest_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

const (
	migrationsTable     = "test_migrations"
	migrationLocksTable = "test_migration_locks"
)

func cleanupMigrations(tb testing.TB, ctx context.Context, db *bun.DB) {
	tb.Cleanup(func() {
		var err error
		_, err = db.NewDropTable().ModelTableExpr(migrationsTable).Exec(ctx)
		require.NoError(tb, err, "drop %q table", migrationsTable)

		_, err = db.NewDropTable().ModelTableExpr(migrationLocksTable).Exec(ctx)
		require.NoError(tb, err, "drop %q table", migrationLocksTable)
	})
}

func TestMigrate(t *testing.T) {
	type Test struct {
		run func(t *testing.T, db *bun.DB)
	}

	tests := []Test{
		{run: testMigrateUpAndDown},
		{run: testMigrateUpError},
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		cleanupMigrations(t, ctx, db)

		for _, test := range tests {
			t.Run(funcName(test.run), func(t *testing.T) {
				test.run(t, db)
			})
		}
	})
}

func testMigrateUpAndDown(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	var history []string

	migrations := migrate.NewMigrations()
	migrations.Add(migrate.Migration{
		Name: "20060102150405",
		Up: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "up1")
			return nil
		},
		Down: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "down1")
			return nil
		},
	})
	migrations.Add(migrate.Migration{
		Name: "20060102160405",
		Up: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "up2")
			return nil
		},
		Down: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "down2")
			return nil
		},
	})

	m := migrate.NewMigrator(db, migrations,
		migrate.WithTableName(migrationsTable),
		migrate.WithLocksTableName(migrationLocksTable),
	)
	err := m.Reset(ctx)
	require.NoError(t, err)

	group, err := m.Migrate(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), group.ID)
	require.Len(t, group.Migrations, 2)
	require.Equal(t, []string{"up1", "up2"}, history)

	history = nil
	group, err = m.Rollback(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), group.ID)
	require.Len(t, group.Migrations, 2)
	require.Equal(t, []string{"down2", "down1"}, history)
}

func testMigrateUpError(t *testing.T, db *bun.DB) {
	ctx := context.Background()

	var history []string

	migrations := migrate.NewMigrations()
	migrations.Add(migrate.Migration{
		Name: "20060102150405",
		Up: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "up1")
			return nil
		},
		Down: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "down1")
			return nil
		},
	})
	migrations.Add(migrate.Migration{
		Name: "20060102160405",
		Up: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "up2")
			return errors.New("failed")
		},
		Down: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "down2")
			return nil
		},
	})
	migrations.Add(migrate.Migration{
		Name: "20060102170405",
		Up: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "up3")
			return errors.New("failed")
		},
		Down: func(ctx context.Context, db *bun.DB) error {
			history = append(history, "down3")
			return nil
		},
	})

	m := migrate.NewMigrator(db, migrations,
		migrate.WithTableName(migrationsTable),
		migrate.WithLocksTableName(migrationLocksTable),
	)
	err := m.Reset(ctx)
	require.NoError(t, err)

	group, err := m.Migrate(ctx)
	require.Error(t, err)
	require.Equal(t, "failed", err.Error())
	require.Equal(t, int64(1), group.ID)
	require.Len(t, group.Migrations, 2)
	require.Equal(t, []string{"up1", "up2"}, history)

	history = nil
	group, err = m.Rollback(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), group.ID)
	require.Len(t, group.Migrations, 2)
	require.Equal(t, []string{"down2", "down1"}, history)
}

func TestAutoMigrator_Run(t *testing.T) {

	tests := []struct {
		fn func(t *testing.T, db *bun.DB)
	}{
		{testRenameTable},
		{testCreateDropTable},
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for _, tt := range tests {
			t.Run(funcName(tt.fn), func(t *testing.T) {
				tt.fn(t, db)
			})
		}
	})
}

func testRenameTable(t *testing.T, db *bun.DB) {
	type initial struct {
		bun.BaseModel `bun:"table:initial"`
		Foo           int `bun:"foo,notnull"`
	}

	type changed struct {
		bun.BaseModel `bun:"table:changed"`
		Foo           int `bun:"foo,notnull"`
	}

	// Arrange
	ctx := context.Background()
	dbInspector, err := sqlschema.NewInspector(db)
	if err != nil {
		t.Skip(err)
	}
	mustResetModel(t, ctx, db, (*initial)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*changed)(nil))

	m, err := migrate.NewAutoMigrator(db,
		migrate.WithTableNameAuto(migrationsTable),
		migrate.WithLocksTableNameAuto(migrationLocksTable),
		migrate.WithModel((*changed)(nil)))
	require.NoError(t, err)

	// Act
	err = m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state, err := dbInspector.Inspect(ctx)
	require.NoError(t, err)

	tables := state.Tables
	require.Len(t, tables, 1)
	require.Equal(t, "changed", tables[0].Name)
}

func testCreateDropTable(t *testing.T, db *bun.DB) {
	type DropMe struct {
		bun.BaseModel `bun:"table:dropme"`
		Foo           int `bun:"foo,identity"`
	}

	type CreateMe struct {
		bun.BaseModel `bun:"table:createme"`
		Bar           string `bun:",pk,default:gen_random_uuid()"`
		Baz           time.Time
	}

	// Arrange
	ctx := context.Background()
	dbInspector, err := sqlschema.NewInspector(db)
	if err != nil {
		t.Skip(err)
	}
	mustResetModel(t, ctx, db, (*DropMe)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*CreateMe)(nil))

	m, err := migrate.NewAutoMigrator(db,
		migrate.WithTableNameAuto(migrationsTable),
		migrate.WithLocksTableNameAuto(migrationLocksTable),
		migrate.WithModel((*CreateMe)(nil)))
	require.NoError(t, err)

	// Act
	err = m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state, err := dbInspector.Inspect(ctx)
	require.NoError(t, err)

	tables := state.Tables
	require.Len(t, tables, 1)
	require.Equal(t, "createme", tables[0].Name)
}

type Journal struct {
	ISBN  string `bun:"isbn,pk"`
	Title string `bun:"title,notnull"`
	Pages int    `bun:"page_count,notnull,default:0"`
}

type Reader struct {
	Username string `bun:",pk,default:gen_random_uuid()"`
}

type ExternalUsers struct {
	bun.BaseModel `bun:"external.users"`
	Name          string `bun:",pk"`
}

func TestDetector_Diff(t *testing.T) {
	testEachDialect(t, func(t *testing.T, dialectName string, dialect schema.Dialect) {
		for _, tt := range []struct {
			name   string
			states func(testing.TB, context.Context, schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State)
			want   []migrate.Operation
		}{
			{
				name: "1 table renamed, 1 added, 2 dropped",
				states: func(tb testing.TB, ctx context.Context, d schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State) {
					// Database state -------------
					type Subscription struct {
						bun.BaseModel `bun:"table:billing.subscriptions"`
					}
					type Review struct{}

					type Author struct {
						Name string `bun:"name"`
					}

					// Model state -------------
					type JournalRenamed struct {
						bun.BaseModel `bun:"table:journals_renamed"`

						ISBN  string `bun:"isbn,pk"`
						Title string `bun:"title,notnull"`
						Pages int    `bun:"page_count,notnull,default:0"`
					}

					return getState(tb, ctx, d,
							(*Author)(nil),
							(*Journal)(nil),
							(*Review)(nil),
							(*Subscription)(nil),
						), getState(tb, ctx, d,
							(*Author)(nil),
							(*JournalRenamed)(nil),
							(*Reader)(nil),
						)
				},
				want: []migrate.Operation{
					&migrate.RenameTable{
						Schema: dialect.DefaultSchema(),
						From:   "journals",
						To:     "journals_renamed",
					},
					&migrate.CreateTable{
						Model: &Reader{}, // (*Reader)(nil) would be more idiomatic, but schema.Tables
					},
					&migrate.DropTable{
						Schema: "billing",
						Name:   "billing.subscriptions", // TODO: fix once schema is used correctly
					},
					&migrate.DropTable{
						Schema: dialect.DefaultSchema(),
						Name:   "reviews",
					},
				},
			},
			{
				name: "renaming does not work across schemas",
				states: func(tb testing.TB, ctx context.Context, d schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State) {
					// Users have the same columns as the "added" ExternalUsers.
					// However, we should not recognize it as a RENAME, because only models in the same schema can be renamed.
					// Instead, this is a DROP + CREATE case.
					type Users struct {
						bun.BaseModel `bun:"external_users"`
						Name          string `bun:",pk"`
					}

					return getState(tb, ctx, d,
							(*Users)(nil),
						), getState(t, ctx, d,
							(*ExternalUsers)(nil),
						)
				},
				want: []migrate.Operation{
					&migrate.DropTable{
						Schema: dialect.DefaultSchema(),
						Name:   "external_users",
					},
					&migrate.CreateTable{
						Model: &ExternalUsers{},
					},
				},
			},
		} {
			t.Run(funcName(tt.states), func(t *testing.T) {
				ctx := context.Background()
				var d migrate.Detector
				stateDb, stateModel := tt.states(t, ctx, dialect)

				got := d.Diff(stateDb, stateModel).Operations()
				checkEqualChangeset(t, got, tt.want)
			})
		}
	})
}

func checkEqualChangeset(tb testing.TB, got, want []migrate.Operation) {
	tb.Helper()

	// Sort alphabetically to ensure we don't fail because of the wrong order
	sort.Slice(got, func(i, j int) bool {
		return got[i].String() < got[j].String()
	})
	sort.Slice(want, func(i, j int) bool {
		return want[i].String() < want[j].String()
	})

	var cgot, cwant migrate.Changeset
	cgot.Add(got...)
	cwant.Add(want...)

	require.Equal(tb, cwant.String(), cgot.String())
}

func getState(tb testing.TB, ctx context.Context, dialect schema.Dialect, models ...interface{}) sqlschema.State {
	tb.Helper()

	tables := schema.NewTables(dialect)
	tables.Register(models...)

	inspector := sqlschema.NewSchemaInspector(tables)
	state, err := inspector.Inspect(ctx)
	if err != nil {
		tb.Skip("get state: %w", err)
	}
	return state
}
