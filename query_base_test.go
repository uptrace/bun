package bun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryComment(t *testing.T) {
	ctx := context.Background()
	ctxWithComment := ContextWithComment(ctx, "test comment from context")

	q := &SelectQuery{}

	b := q.appendComment(ctx, nil)
	require.Equal(t, "", string(b))

	b = q.appendComment(ctxWithComment, nil)
	require.Equal(t, "/*test comment from context*/ ", string(b))

	qWithComment := q.Comment("test comment from api")
	b = qWithComment.appendComment(ctx, nil)
	require.Equal(t, "/*test comment from api*/ ", string(b))

	b = qWithComment.appendComment(ctxWithComment, nil)
	require.Equal(t, "/*test comment from api*/ ", string(b))

	b = qWithComment.Comment("test with /").appendComment(ctx, nil)
	require.Equal(t, "/*test with /*/ ", string(b))

	b = qWithComment.Comment("test with *").appendComment(ctx, nil)
	require.Equal(t, "/*test with **/ ", string(b))

	b = qWithComment.Comment("test with */ closing").appendComment(ctx, nil)
	require.Equal(t, "/*test with \\*\\/ closing*/ ", string(b))

	b = qWithComment.Comment("test with closing at end */").appendComment(ctx, nil)
	require.Equal(t, "/*test with closing at end \\*\\/*/ ", string(b))
}
