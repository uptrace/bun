package dbtest_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/migrate/sqlschema"
)

func TestRefMap_Update(t *testing.T) {
	for _, tt := range []struct {
		name        string
		fks         []sqlschema.FK
		update      func(rm sqlschema.RefMap) int
		wantUpdated int
		wantFKs     []sqlschema.FK
	}{
		{
			name: "update table reference in all FKs that reference its columns",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("x", "y", "z"),
					To:   sqlschema.C("a", "b", "c"),
				},
				{
					From: sqlschema.C("m", "n", "o"),
					To:   sqlschema.C("a", "b", "d"),
				},
			},
			update: func(rm sqlschema.RefMap) int {
				return rm.UpdateT(sqlschema.T("a", "b"), sqlschema.T("a", "new_b"))
			},
			wantUpdated: 2,
			wantFKs: []sqlschema.FK{ // checking 1 of the 2 updated ones should be enough
				{
					From: sqlschema.C("x", "y", "z"),
					To:   sqlschema.C("a", "new_b", "c"),
				},
			},
		},
		{
			name: "update table reference in FK which points to the same table",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "child"),
					To:   sqlschema.C("a", "b", "parent"),
				},
			},
			update: func(rm sqlschema.RefMap) int {
				return rm.UpdateT(sqlschema.T("a", "b"), sqlschema.T("a", "new_b"))
			},
			wantUpdated: 1,
			wantFKs: []sqlschema.FK{
				{
					From: sqlschema.C("a", "new_b", "child"),
					To:   sqlschema.C("a", "new_b", "parent"),
				},
			},
		},
		{
			name: "update column reference in all FKs which depend on it",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("x", "y", "z"),
					To:   sqlschema.C("a", "b", "c"),
				},
				{
					From: sqlschema.C("a", "b", "c"),
					To:   sqlschema.C("m", "n", "o"),
				},
			},
			update: func(rm sqlschema.RefMap) int {
				return rm.UpdateC(sqlschema.C("a", "b", "c"), "c_new")
			},
			wantUpdated: 2,
			wantFKs: []sqlschema.FK{
				{
					From: sqlschema.C("x", "y", "z"),
					To:   sqlschema.C("a", "b", "c_new"),
				},
			},
		},
		{
			name: "foreign keys defined on multiple columns",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c1", "c2"),
					To:   sqlschema.C("q", "r", "s1", "s2"),
				},
				{
					From: sqlschema.C("m", "n", "o", "p"),
					To:   sqlschema.C("a", "b", "c2"),
				},
			},
			update: func(rm sqlschema.RefMap) int {
				return rm.UpdateC(sqlschema.C("a", "b", "c2"), "x2")
			},
			wantUpdated: 2,
			wantFKs: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c1", "x2"),
					To:   sqlschema.C("q", "r", "s1", "s2"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rm := sqlschema.NewRefMap(tt.fks...)

			n := tt.update(rm)

			require.Equal(t, tt.wantUpdated, n)
			require.Equal(t, tt.wantUpdated, len(rm.Updated()))
			checkHasFK(t, rm, tt.wantFKs...)
		})
	}
}

func checkHasFK(tb testing.TB, rm sqlschema.RefMap, fks ...sqlschema.FK) {
outer:
	for _, want := range fks {
		for _, gotptr := range rm {
			if got := *gotptr; got == want {
				continue outer
			}
		}
		tb.Fatalf("did not find FK%+v", want)
	}
}

func TestRefMap_Delete(t *testing.T) {
	for _, tt := range []struct {
		name        string
		fks         []sqlschema.FK
		del         func(rm sqlschema.RefMap) int
		wantDeleted []sqlschema.FK
	}{
		{
			name: "delete FKs that depend on the table",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c"),
					To:   sqlschema.C("x", "y", "z"),
				},
				{
					From: sqlschema.C("m", "n", "o"),
					To:   sqlschema.C("a", "b", "d"),
				},
				{
					From: sqlschema.C("q", "r", "s"),
					To:   sqlschema.C("w", "w", "w"),
				},
			},
			del: func(rm sqlschema.RefMap) int {
				return rm.DeleteT(sqlschema.T("a", "b"))
			},
			wantDeleted: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c"),
					To:   sqlschema.C("x", "y", "z"),
				},
				{
					From: sqlschema.C("m", "n", "o"),
					To:   sqlschema.C("a", "b", "d"),
				},
			},
		},
		{
			name: "delete FKs that depend on the column",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c"),
					To:   sqlschema.C("x", "y", "z"),
				},
				{
					From: sqlschema.C("q", "r", "s"),
					To:   sqlschema.C("w", "w", "w"),
				},
			},
			del: func(rm sqlschema.RefMap) int {
				return rm.DeleteC(sqlschema.C("a", "b", "c"))
			},
			wantDeleted: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c"),
					To:   sqlschema.C("x", "y", "z"),
				},
			},
		},
		{
			name: "foreign keys defined on multiple columns",
			fks: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c1", "c2"),
					To:   sqlschema.C("q", "r", "s1", "s2"),
				},
				{
					From: sqlschema.C("m", "n", "o", "p"),
					To:   sqlschema.C("a", "b", "c2"),
				},
			},
			del: func(rm sqlschema.RefMap) int {
				return rm.DeleteC(sqlschema.C("a", "b", "c1"))
			},
			wantDeleted: []sqlschema.FK{
				{
					From: sqlschema.C("a", "b", "c1", "c2"),
					To:   sqlschema.C("q", "r", "s1", "s2"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rm := sqlschema.NewRefMap(tt.fks...)

			n := tt.del(rm)

			require.Equal(t, len(tt.wantDeleted), n)
			require.ElementsMatch(t, rm.Deleted(), tt.wantDeleted)
		})
	}
}
