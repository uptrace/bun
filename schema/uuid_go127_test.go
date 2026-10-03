//go:build go1.27

package schema

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/dialect/sqltype"
)

func TestGo127StdUUID(t *testing.T) {
	u := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	raw := [16]byte(u)
	dialect := newNopDialect()
	gen := NewQueryGen(dialect)

	t.Run("scanner", func(t *testing.T) {
		scanner := Scanner(reflect.TypeFor[uuid.UUID]())
		require.NotNil(t, scanner)

		for _, src := range []any{
			"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			"a0eebc999c0b4ef8bb6d6bb9bd380a11",
			"{a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11}",
			"urn:uuid:a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			[]byte("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			raw[:],
			raw,
		} {
			var dest uuid.UUID
			require.NoError(t, scanner(reflect.ValueOf(&dest).Elem(), src),
				"failed scanning %T", src)
			require.Equal(t, u, dest, "wrong value for %T", src)
		}
	})

	t.Run("scanner rejects", func(t *testing.T) {
		scanner := Scanner(reflect.TypeFor[uuid.UUID]())

		// A 16-byte string is not a valid textual UUID: raw bytes are only
		// accepted from []byte and [16]byte sources, matching database/sql.
		for _, src := range []any{"0123456789abcdef", "not-a-uuid", []byte("nope"), 42} {
			dest := u
			err := scanner(reflect.ValueOf(&dest).Elem(), src)
			require.Error(t, err, "expected an error for %T", src)
			require.Equal(t, u, dest, "destination must be untouched for %T", src)
		}
	})

	t.Run("scanner nonaddressable destination", func(t *testing.T) {
		// Scanner(uuid.UUID) is called with the addressable field value, so a
		// nonaddressable destination can only come from a caller that builds
		// the reflect.Value itself. It must fail cleanly, never panic.
		scanner := Scanner(reflect.TypeFor[uuid.UUID]())
		require.Error(t, scanner(reflect.ValueOf(u), "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"))
		require.Error(t, scanner(reflect.ValueOf(u), nil))
	})

	t.Run("scanner null", func(t *testing.T) {
		scanner := Scanner(reflect.TypeFor[uuid.UUID]())

		dest := u
		require.NoError(t, scanner(reflect.ValueOf(&dest).Elem(), nil))
		require.Equal(t, uuid.UUID{}, dest)
	})

	t.Run("ptr scanner", func(t *testing.T) {
		scanner := Scanner(reflect.TypeFor[*uuid.UUID]())
		require.NotNil(t, scanner)

		// nil pointer: allocated, as any other pointer field in Bun.
		var dest *uuid.UUID
		require.NoError(t, scanner(reflect.ValueOf(&dest).Elem(), "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"))
		require.NotNil(t, dest)
		require.Equal(t, u, *dest)

		// raw 16 bytes
		dest = nil
		require.NoError(t, scanner(reflect.ValueOf(&dest).Elem(), raw[:]))
		require.NotNil(t, dest)
		require.Equal(t, u, *dest)

		// NULL resets an allocated pointer to the nil UUID
		require.NoError(t, scanner(reflect.ValueOf(&dest).Elem(), nil))
		require.NotNil(t, dest)
		require.Equal(t, uuid.UUID{}, *dest)

		// NULL on a nil pointer keeps it nil
		var nilDest *uuid.UUID
		require.NoError(t, scanner(reflect.ValueOf(&nilDest).Elem(), nil))
		require.Nil(t, nilDest)
	})

	t.Run("appender", func(t *testing.T) {
		appender := Appender(dialect, reflect.TypeFor[uuid.UUID]())
		require.NotNil(t, appender)
		require.Equal(t, `'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'`,
			string(appender(gen, nil, reflect.ValueOf(u))))

		// nonaddressable values must still work; uuid.UUID's encoders have
		// value receivers.
		require.False(t, reflect.ValueOf(u).CanAddr())
		require.Equal(t, `'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'`,
			string(appender(gen, nil, reflect.ValueOf(u))))

		ptrAppender := Appender(dialect, reflect.TypeFor[*uuid.UUID]())
		require.Equal(t, `'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'`,
			string(ptrAppender(gen, nil, reflect.ValueOf(&u))))

		var nilPtr *uuid.UUID
		require.Equal(t, "NULL", string(ptrAppender(gen, nil, reflect.ValueOf(nilPtr))))
	})

	t.Run("discover sql type", func(t *testing.T) {
		require.Equal(t, sqltype.VarChar, DiscoverSQLType(reflect.TypeFor[uuid.UUID]()))
		require.Equal(t, sqltype.VarChar, DiscoverSQLType(reflect.TypeFor[*uuid.UUID]()))
	})
}
