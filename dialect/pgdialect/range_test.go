package pgdialect

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/schema"
)

func TestRangeScanInt64(t *testing.T) {
	tests := []struct {
		src  string
		want Range[int64]
	}{
		{"[1,5)", Range[int64]{Lower: 1, Upper: 5, LowerBound: '[', UpperBound: ')'}},
		{"[-10,-2)", Range[int64]{Lower: -10, Upper: -2, LowerBound: '[', UpperBound: ')'}},
		{"(,5)", Range[int64]{Upper: 5, UpperBound: ')'}},
		{"[3,)", Range[int64]{Lower: 3, LowerBound: '['}},
		{"empty", NewEmptyRange[int64]()},
	}

	for _, test := range tests {
		t.Run(test.src, func(t *testing.T) {
			var r Range[int64]
			require.NoError(t, r.Scan([]byte(test.src)))
			require.Equal(t, test.want, r)
		})
	}
}

func TestRangeScanFloat64(t *testing.T) {
	var r Range[float64]
	require.NoError(t, r.Scan("[1.5,2.25]"))
	require.Equal(t, Range[float64]{Lower: 1.5, Upper: 2.25, LowerBound: '[', UpperBound: ']'}, r)
}

func TestRangeRoundTripInt64(t *testing.T) {
	in := NewRange[int64](1, 5)
	b, err := in.AppendQuery(schema.NewQueryGen(New()), nil)
	require.NoError(t, err)
	require.Equal(t, "'[1,5)'", string(b))

	var out Range[int64]
	require.NoError(t, out.Scan(b[1:len(b)-1]))
	require.Equal(t, in, out)
}

func TestRangeScanUnsupportedType(t *testing.T) {
	var r Range[int32]
	require.NotPanics(t, func() {
		require.Error(t, r.Scan("[1,5)"))
	})
}
