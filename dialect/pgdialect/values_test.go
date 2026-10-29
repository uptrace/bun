package pgdialect

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/feature"
)

func TestValuesCharacterCasts(t *testing.T) {
	for _, tt := range []struct {
		sqlType string
		cast    string
	}{
		{"varchar(3)", "VARCHAR"},
		{"character varying(3)", "VARCHAR"},
		{"CHARACTER   VARYING ( 3 )", "VARCHAR"},
		{"char(3)", "pg_catalog.BPCHAR"},
		{"character(3)", "pg_catalog.BPCHAR"},
		{"bpchar(3)", "BPCHAR"},
		{"char", "pg_catalog.BPCHAR"},
		{"character", "pg_catalog.BPCHAR"},
		{"varchar(3)[]", "VARCHAR[]"},
		{"char(3)[][]", "pg_catalog.BPCHAR[][]"},
		{"varchar(3)[2]", "VARCHAR[2]"},
		{"pg_catalog.varchar(3)", "pg_catalog.VARCHAR"},
		{"pg_catalog.bpchar(3)[]", "pg_catalog.BPCHAR[]"},
		{"varchar", "varchar"},
		{"bpchar", "bpchar"},
		{"text", "text"},
		{"uuid", "uuid"},
		{"uuid[]", "uuid[]"},
		{"jsonb", "jsonb"},
		{"timestamp(3)", "timestamp(3)"},
		{"numeric(5)", "numeric(5)"},
		{"app.varchar(3)", "app.varchar(3)"},
		{`"char"`, `"char"`},
		{"varchar(0)", "varchar(0)"},
		{"varchar(-1)", "varchar(-1)"},
		{"varchar(10485761)", "varchar(10485761)"},
		{"varchar(999999999999999999999999)", "varchar(999999999999999999999999)"},
	} {
		t.Run(tt.sqlType, func(t *testing.T) {
			typeTag := "type:" + strconv.Quote(tt.sqlType)
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Value",
				Type: reflect.TypeFor[*string](),
				Tag:  reflect.StructTag("bun:" + strconv.Quote(typeTag)),
			}})
			model := reflect.New(typ).Interface()
			db := bun.NewDB(nil, New())
			query, err := db.NewValues(model).AppendQuery(db.QueryGen(), nil)
			require.NoError(t, err)
			require.Equal(t, "VALUES (NULL::"+tt.cast+")", string(query))
			require.Equal(t, tt.sqlType, db.Table(typ).FieldMap["value"].CreateTableSQLType)

			query, err = db.NewValues(model).Value("value", "?::varchar(2)", "abcd").
				AppendQuery(db.QueryGen(), nil)
			require.NoError(t, err)
			require.Equal(t, "VALUES ('abcd'::varchar(2))", string(query))

			db = bun.NewDB(nil, New(WithoutFeature(feature.DoubleColonCast)))
			query, err = db.NewValues(model).AppendQuery(db.QueryGen(), nil)
			require.NoError(t, err)
			require.Equal(t, "VALUES (NULL)", string(query))
		})
	}
}
