package bun

import (
	"testing"

	"github.com/uptrace/bun/schema"
)

func Test_whereBaseQuery_GetWhereFields(t *testing.T) {
	for _, tc := range []struct {
		name           string
		whereBaseQuery whereBaseQuery
		want           []string
	}{
		{
			name: "empty",
		},
		{
			name: "one field",
			whereBaseQuery: whereBaseQuery{
				where: []schema.QueryWithSep{
					{
						QueryWithArgs: schema.QueryWithArgs{
							Query: "test > 1",
						},
					},
				},
			},
			want: []string{"test"},
		},
		{
			name: "more fields",
			whereBaseQuery: whereBaseQuery{
				where: []schema.QueryWithSep{
					{
						QueryWithArgs: schema.QueryWithArgs{
							Query: "test = 1",
						},
					},
					{
						QueryWithArgs: schema.QueryWithArgs{
							Query: "test > 1",
						},
					},
					{
						QueryWithArgs: schema.QueryWithArgs{
							Query: "test < 3",
						},
					},
				},
			},
			want: []string{"test", "test", "test"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := tc.whereBaseQuery.GetWhereFields()
			if len(fields) != len(tc.want) {
				t.Fatal("fields length must be equal")
			}
			for i := range fields {
				if fields[i] != tc.want[i] {
					t.Fatalf("got %s, want %s", fields[i], tc.want[i])
				}
			}
		})
	}

}
