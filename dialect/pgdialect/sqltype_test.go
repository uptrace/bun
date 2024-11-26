package pgdialect

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/migrate/sqlschema"
)

func TestInspectorDialect_CompareType(t *testing.T) {
	d := New()

	t.Run("common types", func(t *testing.T) {
		for _, tt := range []struct {
			typ1, typ2 string
			want       bool
		}{
			{"text", "text", true},     // identical types
			{"bigint", "BIGINT", true}, // case-insensitive

			{sqltype.VarChar, pgTypeVarchar, true},
			{sqltype.VarChar, pgTypeCharacterVarying, true},
			{sqltype.VarChar, pgTypeChar, false},
			{sqltype.VarChar, pgTypeCharacter, false},
			{pgTypeCharacterVarying, pgTypeVarchar, true},
			{pgTypeCharacter, pgTypeChar, true},
			{sqltype.VarChar, pgTypeText, false},
			{pgTypeChar, pgTypeText, false},
			{pgTypeVarchar, pgTypeText, false},

			// SQL standards require that TIMESTAMP be default alias for "TIMESTAMP WITH TIME ZONE"
			{sqltype.Timestamp, pgTypeTimestampTz, true},
			{sqltype.Timestamp, pgTypeTimestampWithTz, true},
			{sqltype.Timestamp, pgTypeTimestamp, true}, // Still, TIMESTAMP == TIMESTAMP
			{sqltype.Timestamp, pgTypeTimeTz, false},
			{pgTypeTimestampTz, pgTypeTimestampWithTz, true},
		} {
			eq := " ~ "
			if !tt.want {
				eq = " !~ "
			}
			t.Run(tt.typ1+eq+tt.typ2, func(t *testing.T) {
				got := d.CompareType(
					&sqlschema.BaseColumn{SQLType: tt.typ1},
					&sqlschema.BaseColumn{SQLType: tt.typ2},
				)
				require.Equal(t, tt.want, got)
			})
		}

	})

	t.Run("custom varchar length", func(t *testing.T) {
		for _, tt := range []struct {
			name       string
			col1, col2 sqlschema.BaseColumn
			want       bool
		}{
			{
				name: "varchars of different length are not equivalent",
				col1: sqlschema.BaseColumn{SQLType: "varchar", VarcharLen: 10},
				col2: sqlschema.BaseColumn{SQLType: "varchar"},
				want: false,
			},
			{
				name: "varchar with no explicit length is equivalent to varchar of default length",
				col1: sqlschema.BaseColumn{SQLType: "varchar", VarcharLen: d.DefaultVarcharLen()},
				col2: sqlschema.BaseColumn{SQLType: "varchar"},
				want: true,
			},
			{
				name: "characters with equal custom length",
				col1: sqlschema.BaseColumn{SQLType: "character varying", VarcharLen: 200},
				col2: sqlschema.BaseColumn{SQLType: "varchar", VarcharLen: 200},
				want: true,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				got := d.CompareType(&tt.col1, &tt.col2)
				require.Equal(t, tt.want, got)
			})
		}
	})
}
