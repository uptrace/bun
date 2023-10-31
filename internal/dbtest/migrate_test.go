package dbtest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
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
	di := getDatabaseInspectorOrSkip(t, db)
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
	state, err := di.Inspect(ctx)
	require.NoError(t, err)

	tables := state.Tables
	require.Len(t, tables, 1)
	require.Equal(t, "changed", tables[0].Name)
}

func TestDetector_Diff(t *testing.T) {
	tests := []struct {
		states     func(testing.TB, context.Context, schema.Dialect) (stateDb schema.State, stateModel schema.State)
		operations []migrate.Operation
	}{
		{
			states: testDetectRenamedTable,
			operations: []migrate.Operation{
				&migrate.RenameTable{
					From: "books",
					To:   "books_renamed",
				},
			},
		},
	}

	testEachDialect(t, func(t *testing.T, dialectName string, dialect schema.Dialect) {
		for _, tt := range tests {
			t.Run(funcName(tt.states), func(t *testing.T) {
				ctx := context.Background()
				var d migrate.Detector
				stateDb, stateModel := tt.states(t, ctx, dialect)

				diff := d.Diff(stateDb, stateModel)

				require.Equal(t, tt.operations, diff.Operations())
			})
		}
	})
}

func testDetectRenamedTable(tb testing.TB, ctx context.Context, dialect schema.Dialect) (s1, s2 schema.State) {
	type Book struct {
		bun.BaseModel

		ISBN  string `bun:"isbn,pk"`
		Title string `bun:"title,notnull"`
		Pages int    `bun:"page_count,notnull,default:0"`
	}

	type Author struct {
		bun.BaseModel
		Name string `bun:"name"`
	}

	type BookRenamed struct {
		bun.BaseModel `bun:"table:books_renamed"`

		ISBN  string `bun:"isbn,pk"`
		Title string `bun:"title,notnull"`
		Pages int    `bun:"page_count,notnull,default:0"`
	}
	return getState(tb, ctx, dialect,
			(*Author)(nil),
			(*Book)(nil),
		), getState(tb, ctx, dialect,
			(*Author)(nil),
			(*BookRenamed)(nil),
		)
}

func getState(tb testing.TB, ctx context.Context, dialect schema.Dialect, models ...interface{}) schema.State {
	tables := schema.NewTables(dialect)
	tables.Register(models...)

	inspector := schema.NewInspector(tables)
	state, err := inspector.Inspect(ctx)
	if err != nil {
		tb.Skip("get state: %w", err)
	}
	return state
}
