package dbtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

// TestIssue1394_DroppedFieldValueOnSliceInsert reproduces issue #1394:
// when inserting a slice of structs, getFields() only inspected the first
// element to decide which fields marshal to DEFAULT (and should therefore
// go to the RETURNING clause instead of the VALUES clause). If a later
// element carried a real value for such a field, it was silently dropped
// from the INSERT and never reached the database.
//
// This test uses only SQLite so it runs without external services.
func TestIssue1394_DroppedFieldValueOnSliceInsert(t *testing.T) {
	ctx := context.Background()

	type Issue1394Model struct {
		bun.BaseModel `bun:"table:issue_1394_models,alias:im"`

		ID    int64  `bun:",pk,autoincrement"`
		Name  string `bun:",notnull"`
		Email string `bun:",notnull,default:'none'"`
	}

	db := sqlite(t)

	mustResetModel(t, ctx, db, (*Issue1394Model)(nil))

	// First element has a zero Email; second element has a real Email.
	// Before the fix, getFields() inspected only the first element, saw
	// Email is NotNull + marshalsToDefault (zero value + SQLDefault), and
	// moved Email to RETURNING — so Email was dropped from the INSERT
	// column list entirely. The second element's "alice@example.com" was
	// silently replaced by the column's DEFAULT ('none').
	models := []Issue1394Model{
		{Name: "zero-email"},
		{Name: "with-email", Email: "alice@example.com"},
	}

	_, err := db.NewInsert().Model(&models).Exec(ctx)
	require.NoError(t, err)

	var got []Issue1394Model
	err = db.NewSelect().Model(&got).OrderExpr("im.id ASC").Scan(ctx)
	require.NoError(t, err)

	require.Len(t, got, 2)
	require.Equal(t, "zero-email", got[0].Name)
	require.Equal(t, "none", got[0].Email,
		"first row had no email, so the column DEFAULT ('none') applies")
	require.Equal(t, "with-email", got[1].Name)
	require.Equal(t, "alice@example.com", got[1].Email,
		"second row's Email must be persisted, not silently dropped (issue #1394)")
}
