package dbtest_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestPGArray(t *testing.T) {
	type Model struct {
		ID    int
		Array []string `bun:",array"`
	}

	db := pg()
	defer db.Close()

	_, err := db.NewDropTable().Model((*Model)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Model((*Model)(nil)).Exec(ctx)
	require.NoError(t, err)

	model1 := &Model{
		ID:    123,
		Array: []string{"one", "two", "three"},
	}
	_, err = db.NewInsert().Model(model1).Exec(ctx)
	require.NoError(t, err)

	model2 := new(Model)
	err = db.NewSelect().Model(model2).Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, model1, model2)

	var strs []string
	err = db.NewSelect().Model((*Model)(nil)).Column("array").Scan(ctx, pgdialect.Array(&strs))
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, strs)
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
	db := pg()
	defer db.Close()

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

	db := pg()
	defer db.Close()

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

func TestPGScanIgnoredField(t *testing.T) {
	type Model struct {
		Array []string `bun:"-,array"`
	}

	db := pg()
	defer db.Close()

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

	db := pg()
	defer db.Close()

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
	db := pg()
	defer db.Close()

	_, err := db.Exec("invalid query")
	require.Error(t, err)
	require.Contains(t, err.Error(), "#42601 syntax error")

	_, err = db.Exec("SELECT 1")
	require.NoError(t, err)
}

func TestPGTransaction(t *testing.T) {
	db := pg()
	defer db.Close()

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
	db := pg()
	defer db.Close()

	type Model struct {
		ID int64
	}

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	var num int64
	_, err = db.NewUpdate().Model(new(Model)).Set("id = NULL").WherePK().Exec(ctx, &num)
	require.Equal(t, sql.ErrNoRows, err)
}
