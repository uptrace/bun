//go:build go1.27

package pgdialect

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestSQLType_UUID(t *testing.T) {
	d := New()
	tables := d.Tables()

	type Model struct {
		UUID       uuid.UUID    `bun:",pk"`
		UUIDPtr    *uuid.UUID   `bun:""`
		UUIDArray  []uuid.UUID  `bun:",array"`
		UUIDPtrArr []*uuid.UUID `bun:",array"`
		UUIDNative uuid.UUID    `bun:"type:uuid"`
		RawArray   [][16]byte   `bun:",array"`
		RawBytes   [16]byte
		Strings    []string
	}

	table := tables.Get(reflect.TypeFor[*Model]())
	require.Equal(t, "VARCHAR", table.FieldMap["uuid"].DiscoveredSQLType)
	require.Equal(t, "VARCHAR[]", table.FieldMap["uuid_array"].DiscoveredSQLType)
	require.Equal(t, "VARCHAR[]", table.FieldMap["uuid_ptr_arr"].DiscoveredSQLType)
	require.Equal(t, "uuid", table.FieldMap["uuid_native"].UserSQLType)
	require.Equal(t, "BYTEA[]", table.FieldMap["raw_array"].DiscoveredSQLType)
	require.Equal(t, "BYTEA", table.FieldMap["raw_bytes"].DiscoveredSQLType)
	require.Equal(t, "JSONB", table.FieldMap["strings"].DiscoveredSQLType)
}
