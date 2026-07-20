package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression test for #1306: scanning JSON into a non-nil interface used to
// panic on dest.Addr() because the interface element is not addressable. The
// scan must fail with an error instead, so the query does not deadlock.
func TestScanJSONUnaddressableInterface(t *testing.T) {
	var value any = map[string]any{"a": float64(1)}
	dest := reflect.ValueOf(&value).Elem()

	fn := Scanner(dest.Type())
	require.NotNil(t, fn)

	require.NotPanics(t, func() {
		err := fn(dest, []byte(`{"b":2}`))
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonaddressable")
	})
}

func TestScanJSONAddressableMap(t *testing.T) {
	var value map[string]any
	dest := reflect.ValueOf(&value).Elem()

	fn := Scanner(dest.Type())
	require.NotNil(t, fn)

	require.NoError(t, fn(dest, []byte(`{"b":2}`)))
	require.Equal(t, map[string]any{"b": float64(2)}, value)
}
