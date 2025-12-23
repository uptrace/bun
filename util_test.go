package bun

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_appendComment(t *testing.T) {
	t.Run("ordinary comment", func(t *testing.T) {
		var res []byte
		c := "comment"

		s := appendComment(res, c)
		require.Equal(t, "/* comment */ ", string(s))
	})

	t.Run("only open sequence", func(t *testing.T) {
		var res []byte
		c := "/* comment"

		s := appendComment(res, c)
		require.Equal(t, "/* /\\* comment */ ", string(s))
	})

	t.Run("only close sequence", func(t *testing.T) {
		var res []byte
		c := "comment */"

		s := appendComment(res, c)
		require.Equal(t, "/* comment *\\/ */ ", string(s))
	})

	t.Run("open and close sequences", func(t *testing.T) {
		var res []byte
		c := "/* comment */"

		s := appendComment(res, c)
		require.Equal(t, "/* /\\* comment *\\/ */ ", string(s))
	})

	t.Run("zero bytes", func(t *testing.T) {
		var res []byte
		c := string([]byte{'*', 0, 0, 0, 0, 0, '/'})

		s := appendComment(res, c)
		require.Equal(t, "/* *\\/ */ ", string(s))
	})
}
