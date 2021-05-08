package dbtest_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestPGArray(t *testing.T) {
	type Model struct {
		ID    int
		Array []string `bun:",array"`
	}

	db := pg(t)

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

	Recipe       *Recipe     `bun:"rel:has-one"`
	RecipeID     int         `bun:",pk"`
	Ingredient   *Ingredient `bun:"rel:has-one"`
	IngredientID int         `bun:",pk"`
}

func TestPGMultiTenant(t *testing.T) {
	db := pg(t)

	db = db.WithArg("tenant", bun.Safe("public"))
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
		_, err := db.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)
	}

	recipe := new(Recipe)
	err := db.NewSelect().Model(recipe).Where("id = 1").Relation("Ingredients").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, recipe.Ingredients, 1)
	require.Equal(t, 1, recipe.Ingredients[0].ID)
}
