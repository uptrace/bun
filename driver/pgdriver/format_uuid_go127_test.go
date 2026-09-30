//go:build go1.27

package pgdriver

import (
	"database/sql/driver"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

// Go 1.27's database/sql already converts a uuid.UUID parameter to its textual
// form before the driver sees it, so the driver formatter does not need a
// uuid.UUID or encoding.TextMarshaler case of its own.
func TestFormatQueryStdUUID(t *testing.T) {
	u := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	const want = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

	converted, err := driver.DefaultParameterConverter.ConvertValue(u)
	require.NoError(t, err)
	require.Equal(t, want, converted, "database/sql converts uuid.UUID to its text form")

	query, err := formatQuery("select $1", namedValues(converted))
	require.NoError(t, err)
	require.Equal(t, "select '"+want+"'", query)

	ptr := &u
	converted, err = driver.DefaultParameterConverter.ConvertValue(ptr)
	require.NoError(t, err)
	require.Equal(t, want, converted)

	var nilPtr *uuid.UUID
	converted, err = driver.DefaultParameterConverter.ConvertValue(nilPtr)
	require.NoError(t, err)
	require.Nil(t, converted, "a nil uuid pointer is SQL NULL")
}
