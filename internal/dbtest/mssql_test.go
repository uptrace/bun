package dbtest_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMssqlMerge(t *testing.T) {
	db := mssql2019(t)
	defer db.Close()

	type Model struct {
		ID int64 `bun:",pk,autoincrement"`

		Name  string
		Value string
	}

	err := db.ResetModel(ctx, (*Model)(nil))
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&Model{Name: "A", Value: "hello"}).Exec(ctx)
	require.NoError(t, err)

	newModels := []*Model{
		{
			Name:  "A",
			Value: "world",
		},
		{
			Name:  "B",
			Value: "test",
		},
	}

	changes := []string{}
	_, err = db.NewMerge().
		Model(&Model{}).
		With("_data", db.NewValues(&newModels)).
		Using("_data").
		On("?TableAlias.name = _data.name").
		When("MATCHED THEN UPDATE SET ?TableAlias.value = _data.value").
		When("NOT MATCHED THEN INSERT (name, value) VALUES (_data.name, _data.value)").
		Returning("$action").
		Exec(ctx, &changes)
	require.NoError(t, err)

	require.Len(t, changes, 2)
	require.Equal(t, "UPDATE", changes[0])
	require.Equal(t, "INSERT", changes[1])

}
