//go:build go1.27

package schema

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestNullZeroStdUUID(t *testing.T) {
	gen := NewQueryGen(newNopDialect())

	// The nil UUID is a zero value: it must be rendered as NULL. uuid.UUID is
	// a [16]byte array, so this must not depend on addressability.
	out, err := NullZero(uuid.Nil()).AppendQuery(gen, nil)
	require.NoError(t, err)
	require.Equal(t, "NULL", string(out))

	// A non-nil UUID is not a zero value and must be rendered as text.
	u := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	out, err = NullZero(u).AppendQuery(gen, nil)
	require.NoError(t, err)
	require.Equal(t, `'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'`, string(out))

	// A pointer to the nil UUID is a non-nil pointer, so it is not "zero" by
	// the pointer rule; only the value form is checked here.
	require.False(t, isZero(&u))

	// The zero checker must not panic for either value.
	require.True(t, isZero(uuid.Nil()))
	require.False(t, isZero(u))
	require.True(t, isZero(uuid.UUID{}))
}
