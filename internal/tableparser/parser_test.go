package tableparser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTable_IsZero(t *testing.T) {
	type fields struct {
		name  string
		alias string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if both name and alias are empty",
			fields: fields{
				name:  "",
				alias: "",
			},
			want: true,
		},
		{
			name: "return false if name is not empty",
			fields: fields{
				name:  "users",
				alias: "",
			},
			want: false,
		},
		{
			name: "return false if alias is not empty",
			fields: fields{
				name:  "",
				alias: "u",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := Table{
				name:  tt.fields.name,
				alias: tt.fields.alias,
			}
			got := table.IsZero()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTable_Name(t *testing.T) {
	type fields struct {
		name  string
		alias string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "return alias if alias is not empty",
			fields: fields{
				name:  "users",
				alias: "u",
			},
			want: "u",
		},
		{
			name: "return name if alias is empty",
			fields: fields{
				name:  "users",
				alias: "",
			},
			want: "users",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := Table{
				name:  tt.fields.name,
				alias: tt.fields.alias,
			}
			got := table.Name()
			require.Equal(t, tt.want, got)
		})
	}

}

func Test_cleanQuery(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "check if the query without alias is trimmed",
			args: args{
				query: " users ",
			},
			want: "users",
		},
		{
			name: "check if the query with alias is trimmed",
			args: args{
				query: " users as u ",
			},
			want: "users as u",
		},
		{
			name: "check if the keyword AS is lowercase",
			args: args{
				query: "users AS u",
			},
			want: "users as u",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanQuery(tt.args.query)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_hasAlias(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "check if the query has alias with keyword",
			args: args{
				query: "users as u",
			},
			want: true,
		},
		{
			name: "check if the query has alias without keyword",
			args: args{
				query: "users u",
			},
			want: true,
		},
		{
			name: "check if the query has no alias",
			args: args{
				query: "users",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAlias(tt.args.query)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parseWithAlias(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name string
		args args
		want Table
	}{
		{
			name: "parse query with keyword",
			args: args{
				query: "users as u",
			},
			want: Table{
				name:  "users",
				alias: "u",
			},
		},
		{
			name: "parse query without keyword",
			args: args{
				query: "users u",
			},
			want: Table{
				name:  "users",
				alias: "u",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWithAlias(tt.args.query)
			require.Equal(t, tt.want, got)
		})
	}
}
