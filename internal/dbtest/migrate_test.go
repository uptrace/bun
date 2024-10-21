package dbtest_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/migrate/sqlschema"
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

// newAutoMigrator creates an AutoMigrator configured to use test migratins/locks tables.
// If the dialect doesn't support schema inspections or migrations, the test will fail with the corresponding error.
func newAutoMigrator(tb testing.TB, db *bun.DB, opts ...migrate.AutoMigratorOption) *migrate.AutoMigrator {
	tb.Helper()

	opts = append(opts,
		migrate.WithTableNameAuto(migrationsTable),
		migrate.WithLocksTableNameAuto(migrationLocksTable),
	)

	m, err := migrate.NewAutoMigrator(db, opts...)
	require.NoError(tb, err)
	return m
}

// inspectDbOrSkip returns a function to inspect the current state of the database.
// It calls tb.Skip() if the current dialect doesn't support database inpection and
// fails the test if the inspector cannot successfully retrieve database state.
func inspectDbOrSkip(tb testing.TB, db *bun.DB) func(context.Context) sqlschema.State {
	tb.Helper()
	inspector, err := sqlschema.NewInspector(db)
	if err != nil {
		tb.Skip(err)
	}
	return func(ctx context.Context) sqlschema.State {
		state, err := inspector.Inspect(ctx)
		require.NoError(tb, err)
		return state
	}
}

func TestAutoMigrator_Run(t *testing.T) {

	tests := []struct {
		fn func(t *testing.T, db *bun.DB)
	}{
		{testRenameTable},
		{testRenamedColumns},
		{testCreateDropTable},
		{testAlterForeignKeys},
		{testCustomFKNameFunc},
		{testForceRenameFK},
		{testRenameColumnRenamesFK},
		{testChangeColumnType_AutoCast},
		{testIdentity},
		{testAddDropColumn},
		// {testUnique},
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
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*initial)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*changed)(nil))
	m := newAutoMigrator(t, db, migrate.WithModel((*changed)(nil)))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
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
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*DropMe)(nil))
	mustDropTableOnCleanup(t, ctx, db, (*CreateMe)(nil))
	m := newAutoMigrator(t, db, migrate.WithModel((*CreateMe)(nil)))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	tables := state.Tables

	require.Len(t, tables, 1)
	require.Equal(t, "createme", tables[0].Name)
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
		OwnerID int64        `bun:",notnull"`
		Owner   *OwnerCommon `bun:"rel:belongs-to,join:owner_id=id"`
		ThingID int64        `bun:",notnull"`
		Thing   *ThingCommon `bun:"rel:belongs-to,join:thing_id=id"`
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

	m := newAutoMigrator(t, db, migrate.WithModel(
		(*ThingCommon)(nil),
		(*OwnerCommon)(nil),
		(*ThingsToOwner)(nil),
	))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	defaultSchema := db.Dialect().DefaultSchema()

	// Crated 2 new constraints
	require.Contains(t, state.FKs, sqlschema.FK{
		From: sqlschema.C(defaultSchema, "things_to_owners", "owner_id"),
		To:   sqlschema.C(defaultSchema, "owners", "id"),
	})
	require.Contains(t, state.FKs, sqlschema.FK{
		From: sqlschema.C(defaultSchema, "things_to_owners", "thing_id"),
		To:   sqlschema.C(defaultSchema, "things", "id"),
	})

	// Dropped the initial one
	require.NotContains(t, state.FKs, sqlschema.FK{
		From: sqlschema.C(defaultSchema, "things", "owner_id"),
		To:   sqlschema.C(defaultSchema, "owners", "id"),
	})
}

func testForceRenameFK(t *testing.T, db *bun.DB) {
	// Database state
	type Owner struct {
		ID int64 `bun:",pk"`
	}

	type OwnedThing struct {
		bun.BaseModel `bun:"table:things"`
		ID            int64 `bun:",pk"`
		OwnerID       int64 `bun:"owner_id,notnull"`

		Owner *Owner `bun:"rel:belongs-to,join:owner_id=id"`
	}

	// Model state
	type Person struct {
		ID int64 `bun:",pk"`
	}

	type PersonalThing struct {
		bun.BaseModel `bun:"table:things"`
		ID            int64 `bun:",pk"`
		PersonID      int64 `bun:"owner_id,notnull"`

		Owner *Person `bun:"rel:belongs-to,join:owner_id=id"`
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)

	mustCreateTableWithFKs(t, ctx, db,
		(*Owner)(nil),
		(*OwnedThing)(nil),
	)
	mustDropTableOnCleanup(t, ctx, db, (*Person)(nil))

	m := newAutoMigrator(t, db,
		migrate.WithModel(
			(*Person)(nil),
			(*PersonalThing)(nil),
		),
		migrate.WithRenameFK(true),
		migrate.WithFKNameFunc(func(fk sqlschema.FK) string {
			return strings.Join([]string{
				fk.From.Table, fk.To.Table, "fkey",
			}, "_")
		}),
	)

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	schema := db.Dialect().DefaultSchema()

	wantName, ok := state.FKs[sqlschema.FK{
		From: sqlschema.C(schema, "things", "owner_id"),
		To:   sqlschema.C(schema, "people", "id"),
	}]
	require.True(t, ok, "expect state.FKs to contain things_people_fkey")
	require.Equal(t, wantName, "things_people_fkey")
}

func testCustomFKNameFunc(t *testing.T, db *bun.DB) {
	// Database state
	type Column struct {
		OID   int64 `bun:",pk"`
		RelID int64 `bun:"attrelid,notnull"`
	}
	type Table struct {
		OID int64 `bun:",pk"`
	}

	// Model state
	type ColumnM struct {
		bun.BaseModel `bun:"table:columns"`
		OID           int64 `bun:",pk"`
		RelID         int64 `bun:"attrelid,notnull"`

		Table *Table `bun:"rel:belongs-to,join:attrelid=oid"`
	}
	type TableM struct {
		bun.BaseModel `bun:"table:tables"`
		OID           int64 `bun:",pk"`
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)

	mustCreateTableWithFKs(t, ctx, db,
		(*Table)(nil),
		(*Column)(nil),
	)

	m := newAutoMigrator(t, db,
		migrate.WithFKNameFunc(func(sqlschema.FK) string { return "test_fkey" }),
		migrate.WithModel(
			(*TableM)(nil),
			(*ColumnM)(nil),
		),
	)

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	fkName := state.FKs[sqlschema.FK{
		From: sqlschema.C(db.Dialect().DefaultSchema(), "columns", "attrelid"),
		To:   sqlschema.C(db.Dialect().DefaultSchema(), "tables", "oid"),
	}]
	require.Equal(t, "test_fkey", fkName)
}

func testRenamedColumns(t *testing.T, db *bun.DB) {
	// Database state
	type Original struct {
		bun.BaseModel `bun:"original"`
		ID            int64 `bun:",pk"`
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
		Count         int64 `bun:",pk"` // renamed column in renamed model
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
	m := newAutoMigrator(t, db, migrate.WithModel(
		(*Model2)(nil),
		(*Renamed)(nil),
	))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)

	require.Len(t, state.Tables, 2)

	var renamed, model2 sqlschema.Table
	for _, tbl := range state.Tables {
		switch tbl.Name {
		case "renamed":
			renamed = tbl
		case "models":
			model2 = tbl
		}
	}

	require.Contains(t, renamed.Columns, "count")
	require.Contains(t, model2.Columns, "second_column")
	require.Contains(t, model2.Columns, "do_not_rename")
}

func testRenameColumnRenamesFK(t *testing.T, db *bun.DB) {
	type TennantBefore struct {
		bun.BaseModel `bun:"table:tennants"`
		ID            int64 `bun:",pk,identity"`
		Apartment     int8
		NeighbourID   int64

		Neighbour *TennantBefore `bun:"rel:has-one,join:neighbour_id=id"`
	}

	type TennantAfter struct {
		bun.BaseModel `bun:"table:tennants"`
		TennantID     int64 `bun:",pk,identity"`
		Apartment     int8
		NeighbourID   int64 `bun:"my_neighbour"`

		Neighbour *TennantAfter `bun:"rel:has-one,join:my_neighbour=tennant_id"`
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustCreateTableWithFKs(t, ctx, db, (*TennantBefore)(nil))
	m := newAutoMigrator(t, db,
		migrate.WithRenameFK(true),
		migrate.WithModel((*TennantAfter)(nil)),
	)

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)

	fkName := state.FKs[sqlschema.FK{
		From: sqlschema.C(db.Dialect().DefaultSchema(), "tennants", "my_neighbour"),
		To:   sqlschema.C(db.Dialect().DefaultSchema(), "tennants", "tennant_id"),
	}]
	require.Equal(t, "tennants_my_neighbour_fkey", fkName)
}

// testChangeColumnType_AutoCast checks type changes which can be type-casted automatically,
// i.e. do not require supplying a USING clause (pgdialect).
func testChangeColumnType_AutoCast(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:table"`

		SmallInt     int32     `bun:"bigger_int,pk,identity"`
		Timestamp    time.Time `bun:"ts"`
		DefaultExpr  string    `bun:"default_expr,default:gen_random_uuid()"`
		EmptyDefault string    `bun:"empty_default"`
		Nullable     string    `bun:"not_null"`
		TypeOverride string    `bun:"type:varchar(100)"`
		// ManyValues    []string  `bun:",array"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:table"`

		BigInt       int64     `bun:"bigger_int,pk,identity"`        // int64 maps to bigint
		Timestamp    time.Time `bun:"ts,default:current_timestamp"`  // has default value now
		DefaultExpr  string    `bun:"default_expr,default:random()"` // different default
		EmptyDefault string    `bun:"empty_default,default:''"`      // '' empty string default
		NotNullable  string    `bun:"not_null,notnull"`              // added NOT NULL
		TypeOverride string    `bun:"type:varchar(200)"`             // new length
		// ManyValues    []string  `bun:",array"`                    // did not change
	}

	wantTables := []sqlschema.Table{
		{
			Schema: db.Dialect().DefaultSchema(),
			Name:   "table",
			Columns: map[string]sqlschema.Column{
				// "new_pk": {
				// 	IsPK:    true,
				// 	SQLType: "bigint",
				// },
				"bigger_int": {
					SQLType:    "bigint",
					IsPK:       true,
					IsIdentity: true,
				},
				"ts": {
					SQLType:      "timestamp",         // FIXME(dyma): convert "timestamp with time zone" to sqltype.Timestamp
					DefaultValue: "current_timestamp", // FIXME(dyma): Convert driver-specific value to common "expressions" (e.g. CURRENT_TIMESTAMP == current_timestamp) OR lowercase all types.
					IsNullable:   true,
				},
				"default_expr": {
					SQLType:      "varchar",
					IsNullable:   true,
					DefaultValue: "random()",
				},
				"empty_default": {
					SQLType:      "varchar",
					IsNullable:   true,
					DefaultValue: "", // NOT "''"
				},
				"not_null": {
					SQLType:    "varchar",
					IsNullable: false,
				},
				"type_override": {
					SQLType:    "varchar",
					IsNullable: true,
					VarcharLen: 200,
				},
				// "many_values": {
				// 	SQLType: "array",
				// },
			},
		},
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigrator(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.Tables)
}

func testIdentity(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:table"`
		A             int64 `bun:",notnull,identity"`
		B             int64
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:table"`
		A             int64 `bun:",notnull"`
		B             int64 `bun:",notnull,identity"`
	}

	wantTables := []sqlschema.Table{
		{
			Schema: db.Dialect().DefaultSchema(),
			Name:   "table",
			Columns: map[string]sqlschema.Column{
				"a": {
					SQLType:    sqltype.BigInt,
					IsIdentity: false, // <- drop IDENTITY
				},
				"b": {
					SQLType:    sqltype.BigInt,
					IsIdentity: true, // <- add IDENTITY
				},
			},
		},
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigrator(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.Tables)
}

func testAddDropColumn(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:table"`
		DoNotTouch    string `bun:"do_not_touch"`
		DropMe        string `bun:"dropme"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:table"`
		DoNotTouch    string `bun:"do_not_touch"`
		AddMe         bool   `bun:"addme"`
	}

	wantTables := []sqlschema.Table{
		{
			Schema: db.Dialect().DefaultSchema(),
			Name:   "table",
			Columns: map[string]sqlschema.Column{
				"do_not_touch": {
					SQLType:    sqltype.VarChar,
					IsNullable: true,
				},
				"addme": {
					SQLType:    sqltype.Boolean,
					IsNullable: true,
				},
			},
		},
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigrator(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.Tables)
}

func testUnique(t *testing.T, db *bun.DB) {
	type TableBefore struct {
		bun.BaseModel `bun:"table:table"`
		FirstName     string `bun:"first_name,unique:full_name"`
		LastName      string `bun:"last_name,unique:full_name"`
		Birthday      string `bun:"birthday,unique"`
	}

	type TableAfter struct {
		bun.BaseModel `bun:"table:table"`
		FirstName     string `bun:"first_name,unique:full_name"`
		MiddleName    string `bun:"middle_name,unique:full_name"` // extend "full_name" unique group
		LastName      string `bun:"last_name,unique:full_name"`
		Birthday      string `bun:"birthday"`     // doesn't have to be unique any more
		Email         string `bun:"email,unique"` // new column, unique
	}

	wantTables := []sqlschema.Table{
		{
			Schema: db.Dialect().DefaultSchema(),
			Name:   "table",
			Columns: map[string]sqlschema.Column{
				"first_name": {
					SQLType:    sqltype.VarChar,
					IsNullable: true,
				},
				"middle_name": {
					SQLType:    sqltype.VarChar,
					IsNullable: true,
				},
				"last_name": {
					SQLType:    sqltype.VarChar,
					IsNullable: true,
				},
				"birthday": {
					SQLType:    sqltype.VarChar,
					IsNullable: true,
				},
				"email": {
					SQLType:    sqltype.VarChar,
					IsNullable: true,
				},
			},
		},
	}

	ctx := context.Background()
	inspect := inspectDbOrSkip(t, db)
	mustResetModel(t, ctx, db, (*TableBefore)(nil))
	m := newAutoMigrator(t, db, migrate.WithModel((*TableAfter)(nil)))

	// Act
	err := m.Run(ctx)
	require.NoError(t, err)

	// Assert
	state := inspect(ctx)
	cmpTables(t, db.Dialect().(sqlschema.InspectorDialect), wantTables, state.Tables)
}

// // TODO: rewrite these tests into AutoMigrator tests, Diff should be moved to migrate/internal package
// func TestDiff(t *testing.T) {
// 	type Journal struct {
// 		ISBN  string `bun:"isbn,pk"`
// 		Title string `bun:"title,notnull"`
// 		Pages int    `bun:"page_count,notnull,default:0"`
// 	}

// 	type Reader struct {
// 		Username string `bun:",pk,default:gen_random_uuid()"`
// 	}

// 	type ExternalUsers struct {
// 		bun.BaseModel `bun:"external.users"`
// 		Name          string `bun:",pk"`
// 	}

// 	// ------------------------------------------------------------------------
// 	type ThingNoOwner struct {
// 		bun.BaseModel `bun:"things"`
// 		ID            int64 `bun:"thing_id,pk"`
// 		OwnerID       int64 `bun:",notnull"`
// 	}

// 	type Owner struct {
// 		ID int64 `bun:",pk"`
// 	}

// 	type Thing struct {
// 		bun.BaseModel `bun:"things"`
// 		ID            int64 `bun:"thing_id,pk"`
// 		OwnerID       int64 `bun:",notnull"`

// 		Owner *Owner `bun:"rel:belongs-to,join:owner_id=id"`
// 	}

// 	testEachDialect(t, func(t *testing.T, dialectName string, dialect schema.Dialect) {
// 		defaultSchema := dialect.DefaultSchema()

// 		for _, tt := range []struct {
// 			name   string
// 			states func(testing.TB, context.Context, schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State)
// 			want   []migrate.Operation
// 		}{
// 			{
// 				name: "1 table renamed, 1 created, 2 dropped",
// 				states: func(tb testing.TB, ctx context.Context, d schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State) {
// 					// Database state -------------
// 					type Subscription struct {
// 						bun.BaseModel `bun:"table:billing.subscriptions"`
// 					}
// 					type Review struct{}

// 					type Author struct {
// 						Name string `bun:"name"`
// 					}

// 					// Model state -------------
// 					type JournalRenamed struct {
// 						bun.BaseModel `bun:"table:journals_renamed"`

// 						ISBN  string `bun:"isbn,pk"`
// 						Title string `bun:"title,notnull"`
// 						Pages int    `bun:"page_count,notnull,default:0"`
// 					}

// 					return getState(tb, ctx, d,
// 							(*Author)(nil),
// 							(*Journal)(nil),
// 							(*Review)(nil),
// 							(*Subscription)(nil),
// 						), getState(tb, ctx, d,
// 							(*Author)(nil),
// 							(*JournalRenamed)(nil),
// 							(*Reader)(nil),
// 						)
// 				},
// 				want: []migrate.Operation{
// 					&migrate.RenameTable{
// 						Schema: defaultSchema,
// 						From:   "journals",
// 						To:     "journals_renamed",
// 					},
// 					&migrate.CreateTable{
// 						Model: &Reader{}, // (*Reader)(nil) would be more idiomatic, but schema.Tables
// 					},
// 					&migrate.DropTable{
// 						Schema: "billing",
// 						Name:   "billing.subscriptions",
// 					},
// 					&migrate.DropTable{
// 						Schema: defaultSchema,
// 						Name:   "reviews",
// 					},
// 				},
// 			},
// 			{
// 				name: "renaming does not work across schemas",
// 				states: func(tb testing.TB, ctx context.Context, d schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State) {
// 					// Users have the same columns as the "added" ExternalUsers.
// 					// However, we should not recognize it as a RENAME, because only models in the same schema can be renamed.
// 					// Instead, this is a DROP + CREATE case.
// 					type Users struct {
// 						bun.BaseModel `bun:"external_users"`
// 						Name          string `bun:",pk"`
// 					}

// 					return getState(tb, ctx, d,
// 							(*Users)(nil),
// 						), getState(t, ctx, d,
// 							(*ExternalUsers)(nil),
// 						)
// 				},
// 				want: []migrate.Operation{
// 					&migrate.DropTable{
// 						Schema: defaultSchema,
// 						Name:   "external_users",
// 					},
// 					&migrate.CreateTable{
// 						Model: &ExternalUsers{},
// 					},
// 				},
// 			},
// 			{
// 				name: "detect new FKs on existing columns",
// 				states: func(t testing.TB, ctx context.Context, d schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State) {
// 					// database state
// 					type LonelyUser struct {
// 						bun.BaseModel   `bun:"table:users"`
// 						Username        string `bun:",pk"`
// 						DreamPetKind    string `bun:"pet_kind,notnull"`
// 						DreamPetName    string `bun:"pet_name,notnull"`
// 						ImaginaryFriend string `bun:"friend"`
// 					}

// 					type Pet struct {
// 						Nickname string `bun:",pk"`
// 						Kind     string `bun:",pk"`
// 					}

// 					// model state
// 					type HappyUser struct {
// 						bun.BaseModel `bun:"table:users"`
// 						Username      string `bun:",pk"`
// 						PetKind       string `bun:"pet_kind,notnull"`
// 						PetName       string `bun:"pet_name,notnull"`
// 						Friend        string `bun:"friend"`

// 						Pet        *Pet       `bun:"rel:has-one,join:pet_kind=kind,join:pet_name=nickname"`
// 						BestFriend *HappyUser `bun:"rel:has-one,join:friend=username"`
// 					}

// 					return getState(t, ctx, d,
// 							(*LonelyUser)(nil),
// 							(*Pet)(nil),
// 						), getState(t, ctx, d,
// 							(*HappyUser)(nil),
// 							(*Pet)(nil),
// 						)
// 				},
// 				want: []migrate.Operation{
// 					&migrate.AddFK{
// 						FK: sqlschema.FK{
// 							From: sqlschema.C(defaultSchema, "users", "pet_kind", "pet_name"),
// 							To:   sqlschema.C(defaultSchema, "pets", "kind", "nickname"),
// 						},
// 						ConstraintName: "users_pet_kind_pet_name_fkey",
// 					},
// 					&migrate.AddFK{
// 						FK: sqlschema.FK{
// 							From: sqlschema.C(defaultSchema, "users", "friend"),
// 							To:   sqlschema.C(defaultSchema, "users", "username"),
// 						},
// 						ConstraintName: "users_friend_fkey",
// 					},
// 				},
// 			},
// 			{
// 				name: "create FKs for new tables",
// 				states: func(t testing.TB, ctx context.Context, d schema.Dialect) (stateDb sqlschema.State, stateModel sqlschema.State) {
// 					return getState(t, ctx, d,
// 							(*ThingNoOwner)(nil),
// 						), getState(t, ctx, d,
// 							(*Owner)(nil),
// 							(*Thing)(nil),
// 						)
// 				},
// 				want: []migrate.Operation{
// 					&migrate.CreateTable{
// 						Model: &Owner{},
// 					},
// 					&migrate.AddFK{
// 						FK: sqlschema.FK{
// 							From: sqlschema.C(defaultSchema, "things", "owner_id"),
// 							To:   sqlschema.C(defaultSchema, "owners", "id"),
// 						},
// 						ConstraintName: "things_owner_id_fkey",
// 					},
// 				},
// 			},
// 			{
// 				name: "drop FKs for dropped tables",
// 				states: func(t testing.TB, ctx context.Context, d schema.Dialect) (sqlschema.State, sqlschema.State) {
// 					stateDb := getState(t, ctx, d, (*Owner)(nil), (*Thing)(nil))
// 					stateModel := getState(t, ctx, d, (*ThingNoOwner)(nil))

// 					// Normally a database state will have the names of the constraints filled in, but we need to mimic that for the test.
// 					stateDb.FKs[sqlschema.FK{
// 						From: sqlschema.C(d.DefaultSchema(), "things", "owner_id"),
// 						To:   sqlschema.C(d.DefaultSchema(), "owners", "id"),
// 					}] = "test_fkey"
// 					return stateDb, stateModel
// 				},
// 				want: []migrate.Operation{
// 					&migrate.DropTable{
// 						Schema: defaultSchema,
// 						Name:   "owners",
// 					},
// 					&migrate.DropFK{
// 						FK: sqlschema.FK{
// 							From: sqlschema.C(defaultSchema, "things", "owner_id"),
// 							To:   sqlschema.C(defaultSchema, "owners", "id"),
// 						},
// 						ConstraintName: "test_fkey",
// 					},
// 				},
// 			},
// 		} {
// 			t.Run(tt.name, func(t *testing.T) {
// 				ctx := context.Background()
// 				stateDb, stateModel := tt.states(t, ctx, dialect)

// 				got := migrate.Diff(stateDb, stateModel).Operations()
// 				checkEqualChangeset(t, got, tt.want)
// 			})
// 		}
// 	})
// }

// func checkEqualChangeset(tb testing.TB, got, want []migrate.Operation) {
// 	tb.Helper()

// 	// Sort alphabetically to ensure we don't fail because of the wrong order
// 	sort.Slice(got, func(i, j int) bool {
// 		return got[i].String() < got[j].String()
// 	})
// 	sort.Slice(want, func(i, j int) bool {
// 		return want[i].String() < want[j].String()
// 	})

// 	var cgot, cwant migrate.Changeset
// 	cgot.Add(got...)
// 	cwant.Add(want...)

// 	require.Equal(tb, cwant.String(), cgot.String())
// }

// func getState(tb testing.TB, ctx context.Context, dialect schema.Dialect, models ...interface{}) sqlschema.State {
// 	tb.Helper()

// 	tables := schema.NewTables(dialect)
// 	tables.Register(models...)

// 	inspector := sqlschema.NewSchemaInspector(tables)
// 	state, err := inspector.Inspect(ctx)
// 	if err != nil {
// 		tb.Skip("get state: %w", err)
// 	}
// 	return state
// }
