package bun

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/schema"
)

func TestListAndTuple(t *testing.T) {
	gen := schema.NewNopQueryGen()

	tests := []struct {
		name string
		q    schema.QueryAppender
		want string
	}{
		{
			name: "List([]int)",
			q:    List([]int{1, 2, 3}),
			want: "1, 2, 3",
		},
		{
			name: "List([]string)",
			q:    List([]string{"foo", "bar"}),
			want: "'foo', 'bar'",
		},
		{
			name: "List([][]byte)",
			q:    List([][]byte{[]byte("hello"), []byte("world")}),
			want: `'\x68656c6c6f', '\x776f726c64'`,
		},
		{
			name: "List([][16]byte)",
			q: List([][16]byte{
				{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8},
			}),
			want: `'\x6ba7b8109dad11d180b400c04fd430c8'`,
		},
		{
			name: "List([][]int) - no recursion",
			q:    List([][]int{{1, 2}, {3, 4}}),
			want: "'[1,2]', '[3,4]'",
		},
		{
			name: "List([]int) empty",
			q:    List([]int{}),
			want: "NULL",
		},
		{
			name: "Tuple([]int)",
			q:    Tuple([]int{1, 2, 3}),
			want: "(1, 2, 3)",
		},
		{
			name: "Tuple([][]int)",
			q:    Tuple([][]int{{1, 2}, {3, 4}}),
			want: "((1, 2), (3, 4))",
		},
		{
			name: "Tuple([][]byte)",
			q:    Tuple([][]byte{[]byte("hello"), []byte("world")}),
			want: `('\x68656c6c6f', '\x776f726c64')`,
		},
		{
			name: "Tuple([][16]byte)",
			q: Tuple([][16]byte{
				{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8},
			}),
			want: `('\x6ba7b8109dad11d180b400c04fd430c8')`,
		},
		{
			name: "Tuple([]int) empty",
			q:    Tuple([]int{}),
			want: "(NULL)",
		},
		{
			name: "Tuple(nil)",
			q:    Tuple(nil),
			want: "(NULL)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.q.AppendQuery(gen, nil)
			require.NoError(t, err)
			got := string(b)
			t.Logf("output: %s", got)

			if tt.want != "" {
				require.Equal(t, tt.want, got)
			}
		})
	}
}
