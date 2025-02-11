package dbtest_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/schema"
)

func TestPostgresArray(t *testing.T) {
	type Model struct {
		ID     int64     `bun:",pk,autoincrement"`
		Array1 []string  `bun:",array"`
		Array2 *[]string `bun:",array"`
		Array3 *[]string `bun:",array"`
		Array4 []*string `bun:",array"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })
	mustResetModel(t, ctx, db, (*Model)(nil))

	str1 := "hello"
	str2 := "world"
	model1 := &Model{
		ID:     123,
		Array1: []string{"one", "two", "three"},
		Array2: &[]string{"hello", "world"},
		Array4: []*string{&str1, &str2},
	}
	_, err := db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1, model2)

	var strs []string
	err = db.NewSelect().Model((*Model)(nil)).
		Column("array1").
		Scan(ctx, pgdialect.Array(&strs))
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, strs)

	err = db.NewSelect().Model((*Model)(nil)).
		Column("array3").
		Scan(ctx, pgdialect.Array(&strs))
	require.NoError(t, err)
	require.Nil(t, strs)

	err = db.NewSelect().Model((*Model)(nil)).
		Column("array4").
		Scan(ctx, pgdialect.Array(&strs))
	require.NoError(t, err)
	require.Equal(t, []string{"hello", "world"}, strs)
}

func TestPostgresArrayQuote(t *testing.T) {
	db := pg(t)
	t.Cleanup(func() { db.Close() })

	wanted := []string{"'", "''", "'''", "\""}
	var strs []string
	err := db.NewSelect().
		ColumnExpr("?::text[]", pgdialect.Array(wanted)).
		Scan(ctx, pgdialect.Array(&strs))
	require.NoError(t, err)
	require.Equal(t, wanted, strs)
}

type Hash [32]byte

func (h *Hash) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Hash", src)
	}
	if len(srcB) != len(h) {
		return fmt.Errorf("can't scan []byte of len %d into Hash, want %d", len(srcB), len(h))
	}
	copy(h[:], srcB)
	return nil
}

func (h Hash) Value() (driver.Value, error) {
	return h[:], nil
}

func TestPostgresArrayValuer(t *testing.T) {
	type Model struct {
		ID    int64  `bun:",pk,autoincrement"`
		Array []Hash `bun:",array"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	model1 := &Model{
		ID:    123,
		Array: []Hash{Hash{}},
	}
	_, err := db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1, model2)
}

type Recipe struct {
	bun.BaseModel `bun:"?tenant.recipes"`

	ID          int           `bun:",pk,autoincrement"`
	Ingredients []*Ingredient `bun:"m2m:?tenant.ingredients_recipes"`
}

type Ingredient struct {
	bun.BaseModel `bun:"?tenant.ingredients"`

	ID      int       `bun:",pk,autoincrement"`
	Recipes []*Recipe `bun:"m2m:?tenant.ingredients_recipes"`
}

type IngredientRecipe struct {
	bun.BaseModel `bun:"?tenant.ingredients_recipes"`

	Recipe       *Recipe     `bun:"rel:belongs-to"`
	RecipeID     int         `bun:",pk"`
	Ingredient   *Ingredient `bun:"rel:belongs-to"`
	IngredientID int         `bun:",pk"`
}

func TestPostgresMultiTenant(t *testing.T) {
	db := pg(t)

	db = db.WithNamedArg("tenant", bun.Safe("public"))
	_ = db.Table(reflect.TypeFor[IngredientRecipe]())

	models := []interface{}{
		(*Recipe)(nil),
		(*Ingredient)(nil),
		(*IngredientRecipe)(nil),
	}
	for _, model := range models {
		mustResetModel(t, ctx, db, model)
	}

	models = []interface{}{
		&Recipe{ID: 1},
		&Ingredient{ID: 1},
		&IngredientRecipe{
			RecipeID:     1,
			IngredientID: 1,
		},
	}
	for _, model := range models {
		res, err := db.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), n)
	}

	recipe := new(Recipe)
	err := db.NewSelect().Model(recipe).Where("id = 1").Relation("Ingredients").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, recipe.Ingredients, 1)
	require.Equal(t, 1, recipe.Ingredients[0].ID)
}

func TestPostgresInsertNoRows(t *testing.T) {
	type User struct {
		ID int64 `bun:",pk,autoincrement"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*User)(nil))

	{
		res, err := db.NewInsert().
			Model(&User{ID: 1}).
			On("CONFLICT DO NOTHING").
			Returning("*").
			Exec(ctx)
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), n)
	}

	{
		res, err := db.NewInsert().
			Model(&User{ID: 1}).
			On("CONFLICT DO NOTHING").
			Returning("*").
			Exec(ctx)
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(0), n)
	}
}

func TestPostgresInsertNoRowsIdentity(t *testing.T) {
	type User struct {
		ID int64 `bun:",pk,identity"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*User)(nil))

	{
		res, err := db.NewInsert().
			Model(&User{ID: 1}).
			On("CONFLICT DO NOTHING").
			Returning("*").
			Exec(ctx)
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), n)
	}

	{
		res, err := db.NewInsert().
			Model(&User{ID: 1}).
			On("CONFLICT DO NOTHING").
			Returning("*").
			Exec(ctx)
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(0), n)
	}
}

func TestPostgresScanonlyField(t *testing.T) {
	type Model struct {
		Array []string `bun:",scanonly,array"`
	}

	db := pg(t)

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("? AS array", pgdialect.Array([]string{"foo", "bar"})).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, []string{"foo", "bar"}, model.Array)

	err = db.NewSelect().
		ColumnExpr("NULL AS array").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, []string(nil), model.Array)
}

func TestPostgresScanUUID(t *testing.T) {
	type Model struct {
		Array []uuid.UUID `bun:"type:uuid[],array"`
	}

	db := pg(t)

	ids := []uuid.UUID{uuid.New(), uuid.New()}

	model := new(Model)
	err := db.NewSelect().
		ColumnExpr("? AS array", pgdialect.Array(ids)).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, ids, model.Array)

	err = db.NewSelect().
		ColumnExpr("? AS array", pgdialect.Array([]uuid.UUID{})).
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{}, model.Array)

	err = db.NewSelect().
		ColumnExpr("NULL AS array").
		Scan(ctx, model)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID(nil), model.Array)
}

func TestPostgresInvalidQuery(t *testing.T) {
	db := pg(t)

	_, err := db.Exec("invalid query")
	require.Error(t, err)
	require.Contains(t, err.Error(), "syntax error")

	_, err = db.Exec("SELECT 1")
	require.NoError(t, err)
}

func TestPostgresTransaction(t *testing.T) {
	db := pg(t)

	type Model struct {
		ID int64 `bun:",pk,autoincrement"`
	}

	_, err := db.NewDropTable().Model((*Model)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Conn(tx).Model((*Model)(nil)).Exec(ctx)
	require.NoError(t, err)
	mustDropTableOnCleanup(t, ctx, db, (*Model)(nil))

	n, err := db.NewSelect().Conn(tx).Model((*Model)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	err = tx.Rollback()
	require.NoError(t, err)

	_, err = db.NewSelect().Model((*Model)(nil)).Count(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestPostgresScanWithoutResult(t *testing.T) {
	type Model struct {
		ID int64 `bun:",pk,autoincrement"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	var num int64
	_, err := db.NewUpdate().Model(new(Model)).Set("id = NULL").Where("id = 0").Exec(ctx, &num)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestPostgresIPNet(t *testing.T) {
	type Model struct {
		Network net.IPNet `bun:"type:inet"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	_, ipv4Net, err := net.ParseCIDR("192.0.2.1/24")
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&Model{Network: *ipv4Net}).Exec(ctx)
	require.NoError(t, err)

	model := new(Model)
	err = db.NewSelect().Model(model).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, *ipv4Net, model.Network)
}

func TestPostgresBytea(t *testing.T) {
	type Model struct {
		Bytes []byte
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	_, err := db.NewInsert().Model(&Model{Bytes: []byte("hello")}).Exec(ctx)
	require.NoError(t, err)

	model := new(Model)
	err = db.NewSelect().Model(model).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), model.Bytes)
}

func TestPostgresByteaArray(t *testing.T) {
	type Model struct {
		BytesSlice [][]byte `bun:"type:bytea[]"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	model1 := &Model{BytesSlice: [][]byte{[]byte("hello"), []byte("world")}}
	_, err := db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1.BytesSlice, model2.BytesSlice)
}

func TestPostgresDate(t *testing.T) {
	db := pg(t)
	t.Cleanup(func() { db.Close() })

	var str string
	err := db.NewSelect().ColumnExpr("'2021-09-15'::date").Scan(ctx, &str)
	require.NoError(t, err)
	require.Equal(t, "2021-09-15", str)

	str = ""
	err = db.NewSelect().ColumnExpr("CURRENT_TIMESTAMP::date").Scan(ctx, &str)
	require.NoError(t, err)
	require.NotZero(t, str)

	var tm time.Time
	err = db.NewSelect().ColumnExpr("CURRENT_TIMESTAMP::date").Scan(ctx, &tm)
	require.NoError(t, err)
	require.NotZero(t, tm)

	var nullTime bun.NullTime
	err = db.NewSelect().ColumnExpr("CURRENT_TIMESTAMP::date").Scan(ctx, &nullTime)
	require.NoError(t, err)
	require.False(t, nullTime.IsZero())
}

func TestPostgresTimetz(t *testing.T) {
	db := pg(t)
	t.Cleanup(func() { db.Close() })

	var tm time.Time
	err := db.NewSelect().ColumnExpr("now()::timetz").Scan(ctx, &tm)
	require.NoError(t, err)
	require.NotZero(t, tm)
}

func TestPostgresTimeArray(t *testing.T) {
	type Model struct {
		ID     int64        `bun:",pk,autoincrement"`
		Array1 []time.Time  `bun:",array"`
		Array2 *[]time.Time `bun:",array"`
		Array3 *[]time.Time `bun:",array"`
		Array4 []*time.Time `bun:",array"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	time1 := time.Now()
	time2 := time.Now().Add(time.Hour)
	time3 := time.Now().AddDate(0, 0, 1)

	model1 := &Model{
		ID:     123,
		Array1: []time.Time{time1, time2, time3},
		Array2: &[]time.Time{time1, time2, time3},
		Array4: []*time.Time{&time1, &time2, &time3},
	}
	_, err := db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, len(model1.Array1), len(model2.Array1))

	var times []time.Time
	err = db.NewSelect().Model((*Model)(nil)).
		Column("array1").
		Scan(ctx, pgdialect.Array(&times))
	require.NoError(t, err)
	require.Equal(t, len(times), len(model1.Array1))

	err = db.NewSelect().Model((*Model)(nil)).
		Column("array2").
		Scan(ctx, pgdialect.Array(&times))
	require.NoError(t, err)
	require.Equal(t, 3, len(*model1.Array2))

	err = db.NewSelect().Model((*Model)(nil)).
		Column("array3").
		Scan(ctx, pgdialect.Array(&times))
	require.NoError(t, err)
	require.Nil(t, times)

	err = db.NewSelect().Model((*Model)(nil)).
		Column("array4").
		Scan(ctx, pgdialect.Array(&times))
	require.NoError(t, err)
	require.Equal(t, 3, len(model1.Array4))
}

func TestPostgresOnConflictDoUpdate(t *testing.T) {
	type Model struct {
		ID        int64 `bun:",pk,autoincrement"`
		UpdatedAt time.Time
	}

	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	model := &Model{ID: 1}

	_, err := db.NewInsert().
		Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("updated_at = now()").
		Returning("id, updated_at").
		Exec(ctx)
	require.NoError(t, err)
	require.Zero(t, model.UpdatedAt)

	for i := 0; i < 2; i++ {
		_, err = db.NewInsert().
			Model(model).
			On("CONFLICT (id) DO UPDATE").
			Set("updated_at = now()").
			Returning("id, updated_at").
			Exec(ctx)
		require.NoError(t, err)
		require.NotZero(t, model.UpdatedAt)
	}
}

func TestPostgresOnConflictDoUpdateIdentity(t *testing.T) {
	type Model struct {
		ID        int64 `bun:",pk,identity"`
		UpdatedAt time.Time
	}

	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	model := &Model{ID: 1}

	_, err := db.NewInsert().
		Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("updated_at = now()").
		Returning("id, updated_at").
		Exec(ctx)
	require.NoError(t, err)
	require.Zero(t, model.UpdatedAt)

	for i := 0; i < 2; i++ {
		_, err = db.NewInsert().
			Model(model).
			On("CONFLICT (id) DO UPDATE").
			Set("updated_at = now()").
			Returning("id, updated_at").
			Exec(ctx)
		require.NoError(t, err)
		require.NotZero(t, model.UpdatedAt)
	}
}

func TestPostgresCopyFromCopyTo(t *testing.T) {
	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	qs := []string{
		"CREATE TEMP TABLE copy_src(n int)",
		"CREATE TEMP TABLE copy_dest(n int)",
		"INSERT INTO copy_src SELECT generate_series(1, 1000)",
	}
	for _, q := range qs {
		_, err := conn.ExecContext(ctx, q)
		require.NoError(t, err)
	}

	var buf bytes.Buffer

	{
		res, err := pgdriver.CopyTo(ctx, conn, &buf, "COPY copy_src TO STDOUT")
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1000), n)
	}

	// buf will be used in multiple places
	bufReader := bytes.NewReader(buf.Bytes())

	{
		res, err := pgdriver.CopyFrom(ctx, conn, bufReader, "COPY copy_dest FROM STDIN")
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1000), n)

		var count int
		err = conn.QueryRowContext(ctx, "SELECT count(*) FROM copy_dest").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1000, count)
	}

	t.Run("corrupted data", func(t *testing.T) {
		buf := bytes.NewBufferString("corrupted,data\nrow,two\r\nrow three")
		_, err := pgdriver.CopyFrom(ctx, conn, buf, "COPY copy_dest FROM STDIN")
		require.Error(t, err)
		require.Equal(t,
			`ERROR: invalid input syntax for type integer: "corrupted,data" (SQLSTATE=22P02)`,
			err.Error(),
		)
	})

	// Going to use buf one more time
	_, err = bufReader.Seek(0, 0)
	require.NoError(t, err)

	t.Run("run in transaction", func(t *testing.T) {
		tblName := "test_copy_from"
		qs := []string{
			"CREATE TEMP TABLE %s (n int)",
			"INSERT INTO %s SELECT generate_series(1, 100)",
		}
		for _, q := range qs {
			_, err := conn.ExecContext(ctx, fmt.Sprintf(q, tblName))
			require.NoError(t, err)
		}

		tx, err := conn.BeginTx(ctx, nil)
		require.NoError(t, err)

		_, err = tx.Exec(fmt.Sprintf("TRUNCATE %s", tblName))
		require.NoError(t, err)

		res, err := pgdriver.CopyFrom(ctx, conn, bufReader, fmt.Sprintf("COPY %s FROM STDIN", tblName))
		require.NoError(t, err)

		n, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1000), n)

		err = tx.Commit()
		require.NoError(t, err)
	})
}

func TestPostgresUUID(t *testing.T) {
	type Model struct {
		ID uuid.UUID `bun:",pk,nullzero,type:uuid,default:uuid_generate_v4()"`
	}

	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	require.NoError(t, err)

	mustResetModel(t, ctx, db, (*Model)(nil))

	model := new(Model)
	_, err = db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)
	require.NotZero(t, model.ID)
}

func TestPostgresHStore(t *testing.T) {
	type Model struct {
		ID     int64              `bun:",pk,autoincrement"`
		Attrs1 map[string]string  `bun:",hstore"`
		Attrs2 *map[string]string `bun:",hstore"`
		Attrs3 *map[string]string `bun:",hstore"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS HSTORE;`)
	require.NoError(t, err)
	mustResetModel(t, ctx, db, (*Model)(nil))

	model1 := &Model{
		ID:     123,
		Attrs1: map[string]string{"one": "two", "three": "four"},
		Attrs2: &map[string]string{"two": "three", "four": "five"},
	}
	_, err = db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1, model2)

	attrs1 := make(map[string]string)
	err = db.NewSelect().Model((*Model)(nil)).
		Column("attrs1").
		Scan(ctx, pgdialect.HStore(&attrs1))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"one": "two", "three": "four"}, attrs1)

	attrs2 := make(map[string]string)
	err = db.NewSelect().Model((*Model)(nil)).
		Column("attrs2").
		Scan(ctx, pgdialect.HStore(&attrs2))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"two": "three", "four": "five"}, attrs2)

	var attrs3 map[string]string
	err = db.NewSelect().Model((*Model)(nil)).
		Column("attrs3").
		Scan(ctx, pgdialect.HStore(&attrs3))
	require.NoError(t, err)
	require.Nil(t, attrs3)
}

func TestPostgresHStoreQuote(t *testing.T) {
	db := pg(t)
	t.Cleanup(func() { db.Close() })

	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS HSTORE;`)
	require.NoError(t, err)

	wanted := map[string]string{"'": "'", "''": "''", "'''": "'''", "\"": "\""}
	m := make(map[string]string)
	err = db.NewSelect().
		ColumnExpr("?::hstore", pgdialect.HStore(wanted)).
		Scan(ctx, pgdialect.HStore(&m))
	require.NoError(t, err)
	require.Equal(t, wanted, m)
}

func TestPostgresHStoreEmpty(t *testing.T) {
	db := pg(t)
	t.Cleanup(func() { db.Close() })

	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS HSTORE;`)
	require.NoError(t, err)

	wanted := map[string]string{}
	m := make(map[string]string)
	err = db.NewSelect().
		ColumnExpr("?::hstore", pgdialect.HStore(wanted)).
		Scan(ctx, pgdialect.HStore(&m))
	require.NoError(t, err)
	require.Equal(t, wanted, m)
}

func TestPostgresSkipupdateField(t *testing.T) {
	type Model struct {
		ID        int64 `bun:",pk,autoincrement"`
		Name      string
		CreatedAt time.Time `bun:",skipupdate"`
	}

	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	createdAt := time.Now().Truncate(time.Minute).UTC()

	model := &Model{ID: 1, Name: "foo", CreatedAt: createdAt}

	_, err := db.NewInsert().Model(model).Exec(ctx)
	require.NoError(t, err)
	require.NotZero(t, model.CreatedAt)

	//
	// update field with tag "skipupdate"
	//
	model.CreatedAt = model.CreatedAt.Add(2 * time.Minute)
	_, err = db.NewUpdate().Model(model).WherePK().Exec(ctx)
	require.NoError(t, err)

	//
	// check
	//
	model_ := new(Model)
	model_.ID = model.ID
	err = db.NewSelect().Model(model_).WherePK().Scan(ctx)
	require.NoError(t, err, "select")
	require.NotEmpty(t, model_)
	require.Equal(t, model.ID, model_.ID)
	require.Equal(t, model.Name, model_.Name)
	require.Equal(t, createdAt.UTC(), model_.CreatedAt.UTC())

	require.NotEqual(t, model.CreatedAt.UTC(), model_.CreatedAt.UTC())
}

type Issue722 struct {
	V []byte
}

func (t *Issue722) Value() (driver.Value, error) {
	return t.V, nil
}

func (t *Issue722) Scan(src any) error {
	if src == nil {
		return nil
	}

	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("unsupported data type: %T", src)
	}

	t.V = bytes
	return nil
}

func TestPostgresCustomTypeBytes(t *testing.T) {
	type Model struct {
		ID   int64       `bun:",pk,autoincrement"`
		Data []*Issue722 `bun:",array,type:bytea[]"`
	}

	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	in := &Model{Data: []*Issue722{{V: []byte("hello")}}}
	_, err := db.NewInsert().Model(in).Exec(ctx)
	require.NoError(t, err)

	out := new(Model)
	err = db.NewSelect().Model(out).Scan(ctx)
	require.NoError(t, err)
}

func TestPostgresMultiRange(t *testing.T) {
	type Model struct {
		ID    int64                           `bun:",pk,autoincrement"`
		Value pgdialect.MultiRange[time.Time] `bun:",multirange,type:tstzmultirange"`
	}

	ctx := context.Background()

	db := pg(t)
	t.Cleanup(func() { db.Close() })

	mustResetModel(t, ctx, db, (*Model)(nil))

	r1 := pgdialect.NewRange(time.Unix(1000, 0), time.Unix(2000, 0))
	r2 := pgdialect.NewRange(time.Unix(5000, 0), time.Unix(6000, 0))
	in := &Model{Value: pgdialect.MultiRange[time.Time]{r1, r2}}
	_, err := db.NewInsert().Model(in).Exec(ctx)
	require.NoError(t, err)

	out := new(Model)
	err = db.NewSelect().Model(out).Scan(ctx)
	require.NoError(t, err)
}

type UserID struct {
	ID string
}

func (u UserID) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	v := []byte(`"` + u.ID + `"`)
	return append(b, v...), nil
}

var _ schema.QueryAppender = (*UserID)(nil)

func (r *UserID) Scan(anySrc any) (err error) {
	src, ok := anySrc.([]byte)
	if !ok {
		return fmt.Errorf("pgdialect: Range can't scan %T", anySrc)
	}

	r.ID = string(src)
	return nil
}

var _ sql.Scanner = (*UserID)(nil)

func TestPostgresJSONB(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type Model struct {
		ID        int64    `bun:",pk,autoincrement"`
		Item      Item     `bun:",type:jsonb"`
		ItemPtr   *Item    `bun:",type:jsonb"`
		Items     []Item   `bun:",type:jsonb"`
		ItemsP    []*Item  `bun:",type:jsonb"`
		ItemsNull []*Item  `bun:",type:jsonb"`
		TextItemA []UserID `bun:"type:text[]"`
	}

	db := pg(t)
	t.Cleanup(func() { db.Close() })
	mustResetModel(t, ctx, db, (*Model)(nil))

	item1 := Item{Name: "one"}
	item2 := Item{Name: "two"}
	uid1 := UserID{ID: "1"}
	uid2 := UserID{ID: "2"}
	model1 := &Model{
		ID:        123,
		Item:      item1,
		ItemPtr:   &item2,
		Items:     []Item{item1, item2},
		ItemsP:    []*Item{&item1, &item2},
		ItemsNull: nil,
		TextItemA: []UserID{uid1, uid2},
	}
	_, err := db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1, model2)

	var items []Item
	err = db.NewSelect().Model((*Model)(nil)).
		Column("items").
		Scan(ctx, pgdialect.Array(&items))
	require.NoError(t, err)
	require.Equal(t, []Item{item1, item2}, items)

	err = db.NewSelect().Model((*Model)(nil)).
		Column("itemsp").
		Scan(ctx, pgdialect.Array(&items))
	require.NoError(t, err)
	require.Equal(t, []Item{item1, item2}, items)

	err = db.NewSelect().Model((*Model)(nil)).
		Column("items_null").
		Scan(ctx, pgdialect.Array(&items))
	require.NoError(t, err)
	require.Equal(t, []Item{}, items)
}
