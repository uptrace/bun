package dbtest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
)

func TestInsertSliceDefaults(t *testing.T) {
	type Row struct {
		bun.BaseModel `bun:"table:slice_insert_defaults"`
		ID            string `bun:",pk"`
		Seq           int    `bun:",notnull,default:7"`
		Label         string `bun:",notnull,default:'default'"`
	}
	for _, tc := range []struct {
		name string
		rows []Row
		want []Row
	}{
		{
			name: "first defaults",
			rows: []Row{{}, {Seq: 2, Label: "second"}},
			want: []Row{{Seq: 7, Label: "default"}, {Seq: 2, Label: "second"}},
		},
		{
			name: "later defaults",
			rows: []Row{{Seq: 2, Label: "first"}, {}},
			want: []Row{{Seq: 2, Label: "first"}, {Seq: 7, Label: "default"}},
		},
		{
			name: "last explicit",
			rows: []Row{{}, {}, {Seq: 3, Label: "last"}},
			want: []Row{{Seq: 7, Label: "default"}, {Seq: 7, Label: "default"}, {Seq: 3, Label: "last"}},
		},
		{
			name: "independent columns",
			rows: []Row{{Label: "first"}, {Seq: 2}},
			want: []Row{{Seq: 7, Label: "first"}, {Seq: 2, Label: "default"}},
		},
		{
			name: "all defaults",
			rows: []Row{{}, {}},
			want: []Row{{Seq: 7, Label: "default"}, {Seq: 7, Label: "default"}},
		},
		{
			name: "all explicit",
			rows: []Row{{Seq: 1, Label: "first"}, {Seq: 2, Label: "second"}},
			want: []Row{{Seq: 1, Label: "first"}, {Seq: 2, Label: "second"}},
		},
		{
			name: "single default",
			rows: []Row{{}},
			want: []Row{{Seq: 7, Label: "default"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, pointers := range []bool{false, true} {
				t.Run(fmt.Sprintf("pointers=%v", pointers), func(t *testing.T) {
					testEachDB(t, func(t *testing.T, _ string, db *bun.DB) {
						ctx := context.Background()
						mustResetModel(t, ctx, db, (*Row)(nil))
						rows := append([]Row(nil), tc.rows...)
						want := append([]Row(nil), tc.want...)
						for i := range rows {
							rows[i].ID = fmt.Sprint(i)
							want[i].ID = rows[i].ID
						}
						var model any = &rows
						if pointers {
							ptrs := make([]*Row, len(rows))
							for i := range rows {
								ptrs[i] = &rows[i]
							}
							model = &ptrs
						}
						_, err := db.NewInsert().Model(model).Exec(ctx)
						require.NoError(t, err)
						var got []Row
						err = db.NewSelect().Model(&got).OrderExpr("id ASC").Scan(ctx)
						require.NoError(t, err)
						require.Equal(t, want, got)
					})
				})
			}
		})
	}
}

func TestInsertSliceDefaultPointers(t *testing.T) {
	type Row struct {
		bun.BaseModel `bun:"table:slice_insert_defaults"`
		ID            string `bun:",pk"`
		Seq           *int   `bun:",notnull,default:7"`
	}
	testEachDB(t, func(t *testing.T, _ string, db *bun.DB) {
		ctx := context.Background()
		mustResetModel(t, ctx, db, (*Row)(nil))
		zero, two := 0, 2
		rows := []Row{{ID: "0"}, {ID: "1", Seq: &zero}, {ID: "2", Seq: &two}}
		wantZero, wantTwo, wantSeven := 0, 2, 7
		want := []Row{{ID: "0", Seq: &wantSeven}, {ID: "1", Seq: &wantZero}, {ID: "2", Seq: &wantTwo}}
		_, err := db.NewInsert().Model(&rows).Exec(ctx)
		require.NoError(t, err)
		var got []Row
		err = db.NewSelect().Model(&got).OrderExpr("id ASC").Scan(ctx)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func TestInsertSliceDefaultsOutput(t *testing.T) {
	type Row struct {
		bun.BaseModel `bun:"table:slice_insert_defaults"`
		ID            int64  `bun:",pk,autoincrement"`
		Seq           int    `bun:",notnull,default:7"`
		Label         string `bun:",notnull,default:'default'"`
	}
	db := mssql2019(t)
	rows := []Row{{}, {Seq: 2, Label: "second"}}
	q := db.NewInsert().Model(&rows)
	query, err := q.AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)
	require.Equal(t, `INSERT INTO "slice_insert_defaults" ("seq", "label") OUTPUT INSERTED."id", INSERTED."seq", INSERTED."label" VALUES (DEFAULT, DEFAULT), (2, N'second')`, string(query))
}
