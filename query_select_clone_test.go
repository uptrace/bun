package bun

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun/schema"
)

// Regression test for #1388: Clone must copy execution state, not only the
// public builder slices.
func TestSelectQueryCloneCopiesExecutionState(t *testing.T) {
	sentinelErr := errors.New("sentinel")
	conn := &DB{}
	pk := &schema.Field{Name: "id"}

	q := &SelectQuery{}
	q.conn = conn
	q.err = sentinelErr
	q.flags = q.flags.Set(deletedFlag)
	q.whereFields = []*schema.Field{pk}
	q.with = []WithQuery{
		{name: "mat", materialized: true},
		{name: "not_mat", notMaterialized: true},
	}

	clone := q.Clone()

	require.True(t, clone.conn == IConn(conn), "conn must be copied")
	require.True(t, clone.err == sentinelErr, "err must be copied")
	require.True(t, clone.flags.Has(deletedFlag), "flags must be copied")
	require.Equal(t, []*schema.Field{pk}, clone.whereFields)
	require.True(t, clone.with[0].materialized, "materialized must be copied")
	require.True(t, clone.with[1].notMaterialized, "notMaterialized must be copied")

	// The whereFields slice must be a copy, not an alias.
	clone.whereFields[0] = &schema.Field{Name: "other"}
	require.True(t, q.whereFields[0] == pk, "whereFields must not alias the original")
}

func TestSelectQueryCloneNilWhereFields(t *testing.T) {
	q := &SelectQuery{}
	require.Nil(t, q.Clone().whereFields)

	var nilQuery *SelectQuery
	require.Nil(t, nilQuery.Clone())
}
