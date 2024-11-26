package dbtest_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/internal/ordered"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/migrate/sqlschema"
)

const (
	migrationsTable     = "test_migrations"
	migrationLocksTable = "test_migration_locks"
)

var migrationsDir = filepath.Join(os.TempDir(), "dbtest")

// cleanupMigrations adds a cleanup function to reset migration tables.
// The reset does not run for skipped tests to avoid unnecessary work.
//
// Usage:
//
//	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
//		cleanupMigrations(t, ctx, db)
//		// some test that may generate migration entries in the db
//	})
func cleanupMigrations(tb testing.TB, ctx context.Context, db *bun.DB) {
	tb.Cleanup(func() {
		if tb.Skipped() {
			return
		}

		m := migrate.NewMigrator(db, migrate.NewMigrations(),
			migrate.WithTableName(migrationsTable),
			migrate.WithLocksTableName(migrationLocksTable),
		)
		require.NoError(tb, m.Reset(ctx))
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

// newAutoMigratorOrSkip creates an AutoMigrator configured to use test migratins/locks
// tables and dedicated migrations directory. If an AutoMigrator cannob be created because
// the dialect doesn't support either schema inspections or migrations, the test will be *skipped*
// with the corresponding error.
// Additionally, it will create the migrations directory and if
// one does not exist and add a function to tear it down on cleanup.
func newAutoMigratorOrSkip(tb testing.TB, db *bun.DB, opts ...migrate.AutoMigratorOption) *migrate.AutoMigrator {
	tb.Helper()

	opts = append(opts,
		migrate.WithTableNameAuto(migrationsTable),
		migrate.WithLocksTableNameAuto(migrationLocksTable),
		migrate.WithMigrationsDirectoryAuto(migrationsDir),
	)

	m, err := migrate.NewAutoMigrator(db, opts...)
	if err != nil {
		tb.Skip(err)
	}

	err = os.MkdirAll(migrationsDir, os.ModePerm)
	require.NoError(tb, err, "cannot continue test without migrations directory")

	tb.Cleanup(func() {
		if err := os.RemoveAll(migrationsDir); err != nil {
			tb.Logf("cleanup: remove migrations dir: %v", err)
		}
	})

	return m
}

// inspectDbOrSkip returns a function to inspect the current state of the database.
// The test will be *skipped* if the current dialect doesn't support database inpection
// and fail if the inspector cannot successfully retrieve database state.
func inspectDbOrSkip(tb testing.TB, db *bun.DB, schemaName ...string) func(context.Context) sqlschema.BaseDatabase {
	tb.Helper()

	// For convenience, schemaName is an optional parameter in this function.
	inspectSchema := db.Dialect().DefaultSchema()
	if len(schemaName) > 0 {
		inspectSchema = schemaName[0]
	}

	// AutoMigrator excludes these tables by default, but here we need to do this explicitly.
	inspector, err := sqlschema.NewInspector(db, sqlschema.WithSchemaName(inspectSchema), sqlschema.WithExcludeTables(migrationsTable, migrationLocksTable))
	if err != nil {
		tb.Skip(err)
	}

	return func(ctx context.Context) sqlschema.BaseDatabase {
		state, err := inspector.Inspect(ctx)
		require.NoError(tb, err)
		return state.(sqlschema.BaseDatabase)
	}
}

func TestAutoMigrator_CreateSQLMigrations(t *testing.T) {
	type NewTable struct {
		bun.BaseModel `bun:"table:new_table"`
		Bar           string
		Baz           time.Time
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		ctx := context.Background()
		m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*NewTable)(nil)))

		t.Run("basic", func(t *testing.T) {
			migrations, err := m.CreateSQLMigrations(ctx)
			require.NoError(t, err, "should create migrations successfully")

			require.Len(t, migrations, 2, "expected up/down migration pair")
			require.DirExists(t, migrationsDir)
			checkMigrationFileContains(t, "_auto.up.sql", "CREATE TABLE")
			checkMigrationFileContains(t, "_auto.down.sql", "DROP TABLE")
		})

		t.Run("transactional", func(t *testing.T) {
			migrations, err := m.CreateTxSQLMigrations(ctx)
			require.NoError(t, err, "should create migrations successfully")

			require.Len(t, migrations, 2, "expected up/down migration pair")
			require.DirExists(t, migrationsDir)
			checkMigrationFileContains(t, "_auto.tx.up.sql", "CREATE TABLE", "SET statement_timeout = 0")
			checkMigrationFileContains(t, "_auto.tx.down.sql", "DROP TABLE", "SET statement_timeout = 0")
		})

	})
}

// checkMigrationFileContains expected SQL snippet.
func checkMigrationFileContains(t *testing.T, fileSuffix string, snippets ...string) {
	t.Helper()

	files, err := os.ReadDir(migrationsDir)
	require.NoErrorf(t, err, "list files in %s", migrationsDir)

	for _, f := range files {
		if strings.HasSuffix(f.Name(), fileSuffix) {
			b, err := os.ReadFile(filepath.Join(migrationsDir, f.Name()))
			require.NoError(t, err)
			for _, content := range snippets {
				require.Containsf(t, string(b), content, "expected %s file to contain string", f.Name())
			}
			return
		}
	}
	t.Errorf("no *%s file in migrations directory (%s)", fileSuffix, migrationsDir)
}

// checkMigrationFilesExist checks both up- and down- SQL migration files were created.
func checkMigrationFilesExist(t *testing.T) {
	t.Helper()

	files, err := os.ReadDir(migrationsDir)
	require.NoErrorf(t, err, "list files in %s", migrationsDir)

	var up, down bool
	for _, f := range files {
		if !up && strings.HasSuffix(f.Name(), ".up.sql") {
			up = true
		} else if !down && strings.HasSuffix(f.Name(), ".down.sql") {
			down = true
		}
	}

	if !up {
		t.Errorf("no .up.sql file created in migrations directory (%s)", migrationsDir)
	}
	if !down {
		t.Errorf("no .down.sql file created in migrations directory (%s)", migrationsDir)
	}
}

// runMigrations is a test helper to run AutoMigrator.Migrate(), check that it completed without error and migration files were created.
func runMigrations(t *testing.T, m *migrate.AutoMigrator) {
	t.Helper()

	_, err := m.Migrate(ctx)
	require.NoError(t, err, "auto migration failed")
	checkMigrationFilesExist(t)
}

func TestAutoMigrator_Migrate(t *testing.T) {

	tests := []struct {
		fn func(t *testing.T, db *bun.DB)
	}{
		{testRenameTable},
		{testRenamedColumns},
		{testCreateDropTable},
		{testAlterForeignKeys},
		{testChangeColumnType_AutoCast},
		{testIdentity},
		{testAddDropColumn},
		{testUnique},
		{testUniqueRenamedTable},
		{testUpdatePrimaryKeys},
		{testNothingToMigrate},
	}

	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		for _, tt := range tests {
			t.Run(funcName(tt.fn), func(t *testing.T) {
				// Because they are executed so fast, tests may generate migrations
				// with the same timestamp, so that only the first of them will apply.
				// To eliminate these side-effects we cleanup migration tables after
				// after every test case.
				cleanupMigrations(t, ctx, db)
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
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*initial)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*changed)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*changed)(nil)))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	tables := state.Tables
	require.Equal(t, 1, tables.Len())
	_, found := tables.Load("changed")
	require.True(t, found)
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
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*DropMe)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*CreateMe)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*CreateMe)(nil)))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	tables := state.Tables
	require.Equal(t, 1, tables.Len())
	_, found := tables.Load("createme")
	require.True(t, found)
}

func testAlterForeignKeys(t *testing.T, db *bun.DB) {
	// Initial state -- each thing has one owner
	type OwnerExclusive struct {
		bun.BaseModel `bun:"owners"`
		ID            int64 `bun:",pk"`
	}

	type ThingExclusive struct {
		bun.BaseModel `bun:"things"`
		ID            int64 `bun:",pk"`
		OwnerID       int64 `bun:",notnull"`

		Owner *OwnerExclusive `bun:"rel:belongs-to,join:owner_id=id"`
	}

	// Change -- each thing has multiple owners

	type ThingCommon struct {
		bun.BaseModel `bun:"things"`
		ID            int64 `bun:",pk"`
	}

	type OwnerCommon struct {
		bun.BaseModel `bun:"owners"`
		ID            int64          `bun:",pk"`
		Things        []*ThingCommon `bun:"m2m:things_to_owners,join:Owner=Thing"`
	}

	type ThingsToOwner struct {
		bun.BaseModel `bun:"things_to_owners"`
		OwnerID       int64        `bun:",notnull"`
		Owner         *OwnerCommon `bun:"rel:belongs-to,join:owner_id=id"`
		ThingID       int64        `bun:",notnull"`
		Thing         *ThingCommon `bun:"rel:belongs-to,join:thing_id=id"`
	}

	// Arrange
	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	db.RegisterModel((*ThingsToOwner)(nil))

	mustCreateTableWithFKs(t, ctx, db,
		(*OwnerExclusive)(nil),
		(*ThingExclusive)(nil),
	)
	mustDropTableOnCleanup(t, ctx, db, (*ThingsToOwner)(nil))

	m := newAutoMigratorOrSkip(t, db, migrate.WithModel(
		(*ThingCommon)(nil),
		(*OwnerCommon)(nil),
		(*ThingsToOwner)(nil),
	))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)

	// Crated 2 new constraints
	require.Contains(t, state.ForeignKeys, sqlschema.ForeignKey{
		From: sqlschema.NewColumnReference("things_to_owners", "owner_id"),
		To:   sqlschema.NewColumnReference("owners", "id"),
	}, "expected new FK constraint things_to_owners.owner_id -> owners.id")
	require.Contains(t, state.ForeignKeys, sqlschema.ForeignKey{
		From: sqlschema.NewColumnReference("things_to_owners", "thing_id"),
		To:   sqlschema.NewColumnReference("things", "id"),
	}, "expected new FK constraint things_to_owners.thing_id -> things.id")

	// Dropped the initial one
	require.NotContains(t, state.ForeignKeys, sqlschema.ForeignKey{
		From: sqlschema.NewColumnReference("things", "owner_id"),
		To:   sqlschema.NewColumnReference("owners", "id"),
	}, "expected FK constraint things.owner_id -> owners.id to be dropped")
}

func testRenamedColumns(t *testing.T, db *bun.DB) {
	// Database state
	type Original struct {
		bun.BaseModel `bun:"original"`
		ID            int64 `bun:"id,pk"`
	}

	type Model1 struct {
		bun.BaseModel `bun:"models"`
		ID            string `bun:",pk"`
		DoNotRename   string `bun:",default:2"`
		ColumnTwo     int    `bun:",default:2"`
	}

	// Model state
	type Renamed struct {
		bun.BaseModel `bun:"renamed"`
		Count         int64 `bun:"count,pk"` // renamed column in renamed model
	}

	type Model2 struct {
		bun.BaseModel `bun:"models"`
		ID            string `bun:",pk"`
		DoNotRename   string `bun:",default:2"`
		SecondColumn  int    `bun:",default:2"` // renamed column
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db,
		(*Original)(nil),
		(*Model1)(nil),
	)
	mustDropTableOnCleanup(t, ctx, db, (*Renamed)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel(
		(*Model2)(nil),
		(*Renamed)(nil),
	))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	require.Equal(t, 2, state.Tables.Len())

	var renamed, model2 sqlschema.Table
	for _, tbl := range state.Tables.Values() {
		switch tbl.GetName() {
		case "renamed":
			renamed = tbl
		case "models":
			model2 = tbl
		}
	}

	require.NotNil(t, renamed.GetColumns().Value("count"))
	require.NotNil(t, model2.GetColumns().Value("second_column"))
	require.NotNil(t, model2.GetColumns().Value("do_not_rename"))
}

// testChangeColumnType_AutoCast checks type changes which can be type-casted automatically,
// i.e. do not require supplying a USING clause (pgdialect).
func testChangeColumnType_AutoCast(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:change_me_own_type"`

		SmallInt     int32     `bun:"bigger_int,pk,identity"`
		Timestamp    time.Time `bun:"ts"`
		DefaultExpr  string    `bun:"default_expr,default:gen_random_uuid()"`
		EmptyDefault string    `bun:"empty_default"`
		Nullable     string    `bun:"not_null"`
		TypeOverride string    `bun:"type:varchar(100)"`
		// ManyValues    []string  `bun:",array"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:change_me_own_type"`

		BigInt       int64     `bun:"bigger_int,pk,identity"`        // int64 maps to bigint
		Timestamp    time.Time `bun:"ts,default:current_timestamp"`  // has default value now
		DefaultExpr  string    `bun:"default_expr,default:random()"` // different default
		EmptyDefault string    `bun:"empty_default,default:''"`      // '' empty string default
		NotNullable  string    `bun:"not_null,notnull"`              // added NOT NULL
		TypeOverride string    `bun:"type:varchar(200)"`             // new length
		// ManyValues    []string  `bun:",array"`                    // did not change
	}

	wantTables := ordered.NewMap[string, sqlschema.Table](
		ordered.Pair[string, sqlschema.Table]{
			Key: "change_me_own_type",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "change_me_own_type",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "bigger_int",
						Value: &sqlschema.BaseColumn{
							SQLType:    "bigint",
							IsIdentity: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "ts",
						Value: &sqlschema.BaseColumn{
							SQLType:      "timestamp",         // FIXME(dyma): convert "timestamp with time zone" to sqltype.Timestamp
							DefaultValue: "current_timestamp", // FIXME(dyma): Convert driver-specific value to common "expressions" (e.g. CURRENT_TIMESTAMP == current_timestamp) OR lowercase all types.
							IsNullable:   true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "default_expr",
						Value: &sqlschema.BaseColumn{
							SQLType:      "varchar",
							IsNullable:   true,
							DefaultValue: "random()",
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "empty_default",
						Value: &sqlschema.BaseColumn{
							SQLType:      "varchar",
							IsNullable:   true,
							DefaultValue: "", // NOT "''"
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "not_null",
						Value: &sqlschema.BaseColumn{
							SQLType:    "varchar",
							IsNullable: false,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "type_override",
						Value: &sqlschema.BaseColumn{
							SQLType:    "varchar",
							IsNullable: true,
							VarcharLen: 200,
						},
					},
					// ordered.Pair[string, sqlschema.Column]{
					// 	Key: "many_values",
					// 	Value: &sqlschema.BaseColumn{
					// 		SQLType: "array",
					// 	},
					// },
				),
				PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("bigger_int")},
			},
		},
	)

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.GetTables())
}

func testIdentity(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:bourne_identity"`
		A             int64 `bun:",notnull,identity"`
		B             int64
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:bourne_identity"`
		A             int64 `bun:",notnull"`
		B             int64 `bun:",notnull,identity"`
	}

	wantTables := ordered.NewMap[string, sqlschema.Table](
		ordered.Pair[string, sqlschema.Table]{
			Key: "bourne_identity",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "bourne_identity",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "a",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.BigInt,
							IsIdentity: false, // <- drop IDENTITY
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "b",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.BigInt,
							IsIdentity: true, // <- add IDENTITY
						},
					},
				),
			},
		},
	)

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.GetTables())
}

func testAddDropColumn(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:column_madness"`
		DoNotTouch    string `bun:"do_not_touch"`
		DropMe        string `bun:"dropme"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:column_madness"`
		DoNotTouch    string `bun:"do_not_touch"`
		AddMe         bool   `bun:"addme"`
	}

	wantTables := ordered.NewMap[string, sqlschema.Table](
		ordered.Pair[string, sqlschema.Table]{
			Key: "column_madness",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "column_madness",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "do_not_touch",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "addme",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.Boolean,
							IsNullable: true,
						},
					},
				),
			},
		},
	)

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.GetTables())
}

func testUnique(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:uniqlo_stores"`
		FirstName     string `bun:"first_name,unique:full_name"`
		LastName      string `bun:"last_name,unique:full_name"`
		Birthday      string `bun:"birthday,unique"`
		PetName       string `bun:"pet_name,unique:pet"`
		PetBreed      string `bun:"pet_breed,unique:pet"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:uniqlo_stores"`
		FirstName     string `bun:"first_name,unique:full_name"`
		MiddleName    string `bun:"middle_name,unique:full_name"` // extend "full_name" unique group
		LastName      string `bun:"last_name,unique:full_name"`

		Birthday string `bun:"birthday"`     // doesn't have to be unique any more
		Email    string `bun:"email,unique"` // new column, unique

		PetName  string `bun:"pet_name,unique"`
		PetBreed string `bun:"pet_breed"` // shrink "pet" unique group
	}

	wantTables := ordered.NewMap[string, sqlschema.Table](
		ordered.Pair[string, sqlschema.Table]{
			Key: "uniqlo_stores",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "uniqlo_stores",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "first_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "middle_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "last_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "birthday",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "email",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "pet_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "pet_breed",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
				),
				UniqueConstraints: []sqlschema.Unique{
					{Columns: sqlschema.NewColumns("email")},
					{Columns: sqlschema.NewColumns("pet_name")},
					// We can only be sure of the user-defined index name
					{Name: "full_name", Columns: sqlschema.NewColumns("first_name", "middle_name", "last_name")},
				},
			},
		},
	)

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.GetTables())
}

func testUniqueRenamedTable(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:automigrate.before"`
		FirstName     string `bun:"first_name,unique:full_name"`
		LastName      string `bun:"last_name,unique:full_name"`
		Birthday      string `bun:"birthday,unique"`
		PetName       string `bun:"pet_name,unique:pet"`
		PetBreed      string `bun:"pet_breed,unique:pet"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:automigrate.after"`
		// Expand full_name unique group and rename it.
		FirstName string `bun:"first_name,unique:birth_certificate"`
		LastName  string `bun:"last_name,unique:birth_certificate"`
		Birthday  string `bun:"birthday,unique:birth_certificate"`

		// pet_name and pet_breed have their own unique indices now.
		PetName  string `bun:"pet_name,unique"`
		PetBreed string `bun:"pet_breed,unique"`
	}

	wantTables := ordered.NewMap[string, sqlschema.Table](
		ordered.Pair[string, sqlschema.Table]{
			Key: "after",
			Value: &sqlschema.BaseTable{
				Schema: "automigrate",
				Name:   "after",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "first_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "last_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "birthday",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "pet_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "pet_breed",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
				),
				UniqueConstraints: []sqlschema.Unique{
					{Columns: sqlschema.NewColumns("pet_name")},
					{Columns: sqlschema.NewColumns("pet_breed")},
					{Name: "full_name", Columns: sqlschema.NewColumns("first_name", "last_name", "birthday")},
				},
			},
		},
	)

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db, "automigrate")
	mustCreateSchema(t, ctx, db, "automigrate")
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*TableAfter)(nil))
	m := newAutoMigratorOrSkip(t, db,
		migrate.WithModel((*TableAfter)(nil)),
		migrate.WithSchemaName("automigrate"),
	)

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.Tables)
}

func testUpdatePrimaryKeys(t *testing.T, db *bun.DB) {
	// Has a composite primary key.
	type DropPKBefore struct {
		bun.BaseModel `bun:"table:drop_your_pks"`
		FirstName     string `bun:"first_name,pk"`
		LastName      string `bun:"last_name,pk"`
	}

	// This table doesn't have any primary keys at all.
	type AddNewPKBefore struct {
		bun.BaseModel `bun:"table:add_new_pk"`
		FirstName     string `bun:"first_name"`
		LastName      string `bun:"last_name"`
	}

	// Has an (identity) ID column as primary key.
	type ChangePKBefore struct {
		bun.BaseModel `bun:"table:change_pk"`
		ID            int64  `bun:"deprecated,pk,identity"`
		FirstName     string `bun:"first_name"`
		LastName      string `bun:"last_name"`
	}

	// ------------------------

	// Doesn't have any primary keys.
	type DropPKAfter struct {
		bun.BaseModel `bun:"table:drop_your_pks"`
		FirstName     string `bun:"first_name,notnull"`
		LastName      string `bun:"last_name,notnull"`
	}

	// Has a new (identity) ID column as primary key.
	type AddNewPKAfter struct {
		bun.BaseModel `bun:"table:add_new_pk"`
		ID            int64  `bun:"new_id,pk,identity"`
		FirstName     string `bun:"first_name"`
		LastName      string `bun:"last_name"`
	}

	// Has a composite primary key in place of the old ID.
	type ChangePKAfter struct {
		bun.BaseModel `bun:"table:change_pk"`
		FirstName     string `bun:"first_name,pk"`
		LastName      string `bun:"last_name,pk"`
	}

	wantTables := ordered.NewMap[string, sqlschema.Table](
		ordered.Pair[string, sqlschema.Table]{
			Key: "drop_your_pks",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "drop_your_pks",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "first_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: false,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "last_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: false,
						},
					},
				),
			},
		},
		ordered.Pair[string, sqlschema.Table]{
			Key: "add_new_pk",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "add_new_pk",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "new_id",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.BigInt,
							IsNullable: false,
							IsIdentity: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "first_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "last_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: true,
						},
					},
				),
				PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("new_id")},
			},
		},
		ordered.Pair[string, sqlschema.Table]{
			Key: "change_pk",
			Value: &sqlschema.BaseTable{
				Schema: db.Dialect().DefaultSchema(),
				Name:   "change_pk",
				Columns: ordered.NewMap[string, sqlschema.Column](
					ordered.Pair[string, sqlschema.Column]{
						Key: "first_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: false,
						},
					},
					ordered.Pair[string, sqlschema.Column]{
						Key: "last_name",
						Value: &sqlschema.BaseColumn{
							SQLType:    sqltype.VarChar,
							IsNullable: false,
						},
					},
				),
				PrimaryKey: &sqlschema.PrimaryKey{Columns: sqlschema.NewColumns("first_name", "last_name")},
			},
		},
	)

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db,
		(*DropPKBefore)(nil),
		(*AddNewPKBefore)(nil),
		(*ChangePKBefore)(nil),
	)
	m := newAutoMigratorOrSkip(t, db, migrate.WithModel(
		(*DropPKAfter)(nil),
		(*AddNewPKAfter)(nil),
		(*ChangePKAfter)(nil)),
	)

	// Act
	runMigrations(t, m)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.Tables)
}

func testNothingToMigrate(t *testing.T, db *bun.DB) {
	type BoringThing struct {
		AlwaysBlue string `bun:"colour,default:'blue'"`
	}

	ctx := context.Background()
	mustResetModel(t, ctx, db, (*BoringThing)(nil))
	m := newAutoMigratorOrSkip(t, db,
		migrate.WithModel((*BoringThing)(nil)),
	)

	// Act
	_, err := m.Migrate(ctx) // do not use runMigrations because we do not expect any files to be created
	require.NoError(t, err, "auto migration failed")

	migrator := migrate.NewMigrator(db, migrate.NewMigrations(), migrate.WithTableName(migrationsTable))
	applied, err := migrator.AppliedMigrations(ctx)
	require.NoError(t, err, "fetch applied migrations")
	require.Empty(t, applied, "nothing to migrate, AppliedMigrations not empty")
}
