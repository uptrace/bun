package dbtest_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestPostgresValuesCharacterLength(t *testing.T) {
	type Model struct {
		ID               uuid.UUID `bun:",pk,type:uuid"`
		Varchar          string    `bun:"type:varchar(3)"`
		CharacterVarying string    `bun:"type:character varying(3)"`
		Char             string    `bun:"type:char(3)"`
		Character        string    `bun:"type:character(3)"`
		Bpchar           string    `bun:"type:bpchar(3)"`
		CharDefault      string    `bun:"type:char"`
		VarcharArray     []string  `bun:"type:varchar(3)[]"`
		CharArray        []string  `bun:"type:char(3)[]"`
		Nullable         *string   `bun:"type:varchar(3)"`
		Count            int
	}

	newModels := func() []Model {
		old := "old"
		models := make([]Model, 2)
		for i := range models {
			models[i] = Model{
				ID:               uuid.UUID{15: byte(i + 1)},
				Varchar:          "old",
				CharacterVarying: "old",
				Char:             "old",
				Character:        "old",
				Bpchar:           "old",
				CharDefault:      "a",
				VarcharArray:     []string{"old"},
				CharArray:        []string{"old"},
				Count:            i + 1,
			}
		}
		models[1].Nullable = &old
		return models
	}

	tests := []struct {
		column string
		set    func(*Model, string)
		valid  string
	}{
		{"varchar", func(m *Model, s string) { m.Varchar = s }, "界界界"},
		{"character_varying", func(m *Model, s string) { m.CharacterVarying = s }, "界界界"},
		{"char", func(m *Model, s string) { m.Char = s }, "界界界"},
		{"character", func(m *Model, s string) { m.Character = s }, "界界界"},
		{"bpchar", func(m *Model, s string) { m.Bpchar = s }, "界界界"},
		{"char_default", func(m *Model, s string) { m.CharDefault = s }, "界"},
		{"varchar_array", func(m *Model, s string) { m.VarcharArray = []string{"old", s} }, "界界界"},
		{"char_array", func(m *Model, s string) { m.CharArray = []string{"old", s} }, "界界界"},
		{"nullable", func(m *Model, s string) { m.Nullable = &s }, "界界界"},
	}
	for _, driver := range []string{pgName, pgxName} {
		t.Run(driver, func(t *testing.T) {
			db := allDBs[driver](t)
			mustResetModel(t, ctx, db, (*Model)(nil))
			for _, tt := range tests {
				t.Run(tt.column, func(t *testing.T) {
					t.Run("values", func(t *testing.T) {
						models := newModels()
						tt.set(&models[1], "界界界界")
						var got []Model
						err := db.NewSelect().
							With("_data", db.NewValues(&models)).
							Model(&got).
							ModelTableExpr("_data AS model").
							OrderExpr("model.id").
							Scan(ctx)
						require.NoError(t, err)
						require.Equal(t, models, got)
					})
					t.Run("bulk", func(t *testing.T) {
						_, err := db.NewTruncateTable().Model((*Model)(nil)).Exec(ctx)
						require.NoError(t, err)
						models := newModels()
						_, err = db.NewInsert().Model(&models).Exec(ctx)
						require.NoError(t, err)

						tt.set(&models[1], "toolong")
						_, err = db.NewUpdate().Model(&models).Column(tt.column).Bulk().Exec(ctx)
						require.ErrorContains(t, err, "value too long for type")
						var got []Model
						require.NoError(t, db.NewSelect().Model(&got).OrderExpr("model.id").Scan(ctx))
						require.Equal(t, newModels(), got)

						tt.set(&models[1], tt.valid)
						_, err = db.NewUpdate().Model(&models).Column(tt.column).Bulk().Exec(ctx)
						require.NoError(t, err)
						got = nil
						require.NoError(t, db.NewSelect().Model(&got).OrderExpr("model.id").Scan(ctx))
						require.Equal(t, models, got)
					})
				})
			}
			t.Run("char_comparison", func(t *testing.T) {
				models := newModels()
				models[0].Char = "a"
				models[1].Char = "a "
				var count int
				err := db.NewSelect().With("_data", db.NewValues(&models)).
					ColumnExpr("count(*)").TableExpr("_data").
					Where("? = ?::CHAR(3)", bun.Ident("char"), "a").Scan(ctx, &count)
				require.NoError(t, err)
				require.Equal(t, 2, count)
			})
		})
	}
}
