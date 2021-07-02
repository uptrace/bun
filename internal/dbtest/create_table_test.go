package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestCreateTable(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, *bun.DB)
	}{
		{"createWithStringPK", testCreateWithStringPK},
	}

	testEachDB(t, func(t *testing.T, db *bun.DB) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				test.run(t, db)
			})
		}
	})
}

type UserForCreate struct {
	ID   string `bun:",pk"`
	Name string
}

func testCreateWithStringPK(t *testing.T, db *bun.DB) {
	_, err := db.NewCreateTable().
		Model((*UserForCreate)(nil)).
		IfNotExists().
		Exec(context.Background())
	require.NoError(t, err)
}
