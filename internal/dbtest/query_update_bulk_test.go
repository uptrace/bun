package dbtest_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

func TestUpdateBulkRequiresPKs(t *testing.T) {
	type NoPK struct {
		Name  string
		Value string
	}

	type SoftDeleteNoPK struct {
		Name      string
		Value     string
		DeletedAt time.Time `bun:",soft_delete,nullzero"`
	}

	// Neither helper connects to a database until a query is executed.
	for name, newDB := range map[string]func(testing.TB) *bun.DB{"pg": pg, "sqlite": sqlite} {
		db := newDB(t)

		t.Run(name+"/no pk", func(t *testing.T) {
			models := []NoPK{{Name: "a", Value: "1"}}
			_, err := db.NewUpdate().Model(&models).Bulk().AppendQuery(db.QueryGen(), nil)
			require.EqualError(t, err, "bun: model=NoPK does not have primary keys")
		})

		// Without the check, the WHERE clause only filters out soft-deleted
		// rows, so every live row of the table would be updated.
		t.Run(name+"/soft delete no pk", func(t *testing.T) {
			models := []SoftDeleteNoPK{{Name: "a", Value: "1"}}
			_, err := db.NewUpdate().Model(&models).Bulk().AppendQuery(db.QueryGen(), nil)
			require.EqualError(t, err, "bun: model=SoftDeleteNoPK does not have primary keys")
		})
	}
}
