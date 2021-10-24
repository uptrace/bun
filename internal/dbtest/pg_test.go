package dbtest_test

import (
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
)

func TestPGArray(t *testing.T) {
	type Model struct {
		ID     int64
		Array1 []string  `bun:",array"`
		Array2 *[]string `bun:",array"`
		Array3 *[]string `bun:",array"`
	}

	db := pg(t)
	defer db.Close()

	_, err := db.NewDropTable().Model((*Model)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Model((*Model)(nil)).Exec(ctx)
	require.NoError(t, err)

	model1 := &Model{
		ID:     123,
		Array1: []string{"one", "two", "three"},
		Array2: &[]string{"hello", "world"},
	}
	_, err = db.NewInsert().Model(model1).Exec(ctx)
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
}

func TestPGArrayQuote(t *testing.T) {
	db := pg(t)
	defer db.Close()

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

func TestPGArrayValuer(t *testing.T) {
	type Model struct {
		ID    int64
		Array []Hash `bun:",array"`
	}

	db := pg(t)
	defer db.Close()

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	model1 := &Model{
		ID:    123,
		Array: []Hash{Hash{}},
	}
	_, err = db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1, model2)
}

type Recipe struct {
	bun.BaseModel `bun:"?tenant.recipes"`

	ID          int
	Ingredients []*Ingredient `bun:"m2m:?tenant.ingredients_recipes"`
}

type Ingredient struct {
	bun.BaseModel `bun:"?tenant.ingredients"`

	ID      int
	Recipes []*Recipe `bun:"m2m:?tenant.ingredients_recipes"`
}

type IngredientRecipe struct {
	bun.BaseModel `bun:"?tenant.ingredients_recipes"`

	Recipe       *Recipe     `bun:"rel:belongs-to"`
	RecipeID     int         `bun:",pk"`
	Ingredient   *Ingredient `bun:"rel:belongs-to"`
	IngredientID int         `bun:",pk"`
}

func TestPGMultiTenant(t *testing.T) {
	db := pg(t)

	db = db.WithNamedArg("tenant", bun.Safe("public"))
	_ = db.Table(reflect.TypeOf((*IngredientRecipe)(nil)).Elem())

	models := []interface{}{
		(*Recipe)(nil),
		(*Ingredient)(nil),
		(*IngredientRecipe)(nil),
	}
	for _, model := range models {
		_, err := db.NewDropTable().Model(model).IfExists().Exec(ctx)
		require.NoError(t, err)

		_, err = db.NewCreateTable().Model(model).Exec(ctx)
		require.NoError(t, err)
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

func TestPGInsertNoRows(t *testing.T) {
	type User struct {
		ID int64
	}

	db := pg(t)

	err := db.ResetModel(ctx, (*User)(nil))
	require.NoError(t, err)

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

func TestPGScanonlyField(t *testing.T) {
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

func TestPGScanUUID(t *testing.T) {
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

func TestPGInvalidQuery(t *testing.T) {
	db := pg(t)

	_, err := db.Exec("invalid query")
	require.Error(t, err)
	require.Contains(t, err.Error(), "#42601 syntax error")

	_, err = db.Exec("SELECT 1")
	require.NoError(t, err)
}

func TestPGTransaction(t *testing.T) {
	db := pg(t)

	type Model struct {
		ID int64
	}

	_, err := db.NewDropTable().Model((*Model)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Conn(tx).Model((*Model)(nil)).Exec(ctx)
	require.NoError(t, err)

	n, err := db.NewSelect().Conn(tx).Model((*Model)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	err = tx.Rollback()
	require.NoError(t, err)

	_, err = db.NewSelect().Model((*Model)(nil)).Count(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestPGScanWithoutResult(t *testing.T) {
	db := pg(t)
	defer db.Close()

	type Model struct {
		ID int64
	}

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	var num int64
	_, err = db.NewUpdate().Model(new(Model)).Set("id = NULL").Where("id = 0").Exec(ctx, &num)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestPGIPNet(t *testing.T) {
	type Model struct {
		Network net.IPNet `bun:"type:inet"`
	}

	db := pg(t)
	defer db.Close()

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	_, ipv4Net, err := net.ParseCIDR("192.0.2.1/24")
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&Model{Network: *ipv4Net}).Exec(ctx)
	require.NoError(t, err)

	model := new(Model)
	err = db.NewSelect().Model(model).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, *ipv4Net, model.Network)
}

func TestPGBytea(t *testing.T) {
	type Model struct {
		Bytes []byte
	}

	db := pg(t)
	defer db.Close()

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&Model{Bytes: []byte("hello")}).Exec(ctx)
	require.NoError(t, err)

	model := new(Model)
	err = db.NewSelect().Model(model).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), model.Bytes)
}

func TestPGDate(t *testing.T) {
	db := pg(t)
	defer db.Close()

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

func TestPGTimetz(t *testing.T) {
	db := pg(t)
	defer db.Close()

	var tm time.Time
	err := db.NewSelect().ColumnExpr("now()::timetz").Scan(ctx, &tm)
	require.NoError(t, err)
	require.NotZero(t, tm)
}

func TestPGOnConflictDoUpdate(t *testing.T) {
	type Model struct {
		ID        int64
		UpdatedAt time.Time
	}

	ctx := context.Background()

	db := pg(t)
	defer db.Close()

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	model := &Model{ID: 1}

	_, err = db.NewInsert().
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
