package dbtest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
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
