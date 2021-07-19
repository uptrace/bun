package dbtest_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
)

func TestMySQL(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*testing.T, *bun.DB)
	}{
		{"testIntToBool", testIntToBool},
	}

	db := mysql(t)
	if _, ok := os.LookupEnv("DEBUG"); ok {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose()))
	}

	var err error
	_, err = db.NewDropTable().Model((*Hero)(nil)).IfExists().Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewCreateTable().Model((*Hero)(nil)).Exec(ctx)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.fn(t, db)
		})
	}
}

type Hero struct {
	Name        string
	FromRadiant bool
}

func testIntToBool(t *testing.T, db *bun.DB) {
	var err error
	ctx := context.Background()

	herosToInsert := []Hero{
		Hero{
			Name:        "Axe",
			FromRadiant: false,
		},
		Hero{
			Name:        "DragonKnight",
			FromRadiant: true,
		},
	}
	_, err = db.NewInsert().Model(&herosToInsert).Exec(ctx)
	require.NoError(t, err)

	herosSelected := []Hero{}
	err = db.NewSelect().Model(&herosSelected).Order("name asc").Scan(ctx, &herosSelected)
	require.NoError(t, err)
	require.Equal(t, 2, len(herosSelected))
	require.Equal(t, false, herosSelected[0].FromRadiant)
	require.Equal(t, true, herosSelected[1].FromRadiant)

	hero := Hero{}
	err = db.NewSelect().Model(&hero).Where("from_radiant=?", true).Scan(ctx, &hero)
	require.NoError(t, err)
	require.Equal(t, true, hero.FromRadiant)
	require.Equal(t, 2, len(herosSelected))
	require.Equal(t, true, hero.FromRadiant)
}
