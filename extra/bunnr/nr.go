package bunnr

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"
	"github.com/uptrace/bun"
)

type NRBunHook struct {
	baseSegment newrelic.DatastoreSegment
}

type nrBunCtxKey string

const nrBunSegmentKey nrBunCtxKey = "nrbunsegment"

// NewNRHook creates a new bun.QueryHook which reports database usage
// information to newrelic.
func NewNRHook(databaseName, host, portPathOrId string) *NRBunHook {
	return &NRBunHook{
		baseSegment: newrelic.DatastoreSegment{
			Product:      newrelic.DatastorePostgres,
			DatabaseName: databaseName,
			Host:         host,
			PortPathOrID: portPathOrId,
		},
	}
}

func (nbh *NRBunHook) BeforeQuery(ctx context.Context, q *bun.QueryEvent) context.Context {
	segment := nbh.baseSegment

	if q.Model != nil {
		if t, ok := q.Model.(bun.TableModel); ok {
			segment.Operation = q.Operation()
			segment.Collection = t.Table().Name
		} else {
			sqlparse.ParseQuery(&segment, q.Query)
		}
	} else {
		sqlparse.ParseQuery(&segment, q.Query)
	}
	segment.StartTime = newrelic.FromContext(ctx).StartSegmentNow()
	return context.WithValue(ctx, nrBunSegmentKey, &segment)

}
func (nbh *NRBunHook) AfterQuery(ctx context.Context, q *bun.QueryEvent) {
	segment := ctx.Value(nrBunSegmentKey).(*newrelic.DatastoreSegment)
	segment.End()
}
