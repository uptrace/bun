package bunrelic

import (
	"context"

	"github.com/TommyLeng/bun"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"
)

type QueryHook struct {
	baseSegment newrelic.DatastoreSegment
}

var _ bun.QueryHook = (*QueryHook)(nil)

type nrBunCtxKey string

const nrBunSegmentKey nrBunCtxKey = "nrbunsegment"

// NewQueryHook creates a new bun.QueryHook which reports database usage
// information to new relic.
func NewQueryHook(options ...Option) *QueryHook {
	h := &QueryHook{}
	for _, o := range options {
		o(h)
	}
	return h
}

func (q *QueryHook) BeforeQuery(ctx context.Context, qe *bun.QueryEvent) context.Context {
	segment := q.baseSegment

	if qe.Model != nil {
		if t, ok := qe.Model.(bun.TableModel); ok {
			segment.Operation = qe.Operation()
			segment.Collection = t.Table().Name
		} else {
			sqlparse.ParseQuery(&segment, qe.Query)
		}
	} else {
		sqlparse.ParseQuery(&segment, qe.Query)
	}
	segment.StartTime = newrelic.FromContext(ctx).StartSegmentNow()
	return context.WithValue(ctx, nrBunSegmentKey, &segment)

}
func (q *QueryHook) AfterQuery(ctx context.Context, qe *bun.QueryEvent) {
	segment := ctx.Value(nrBunSegmentKey).(*newrelic.DatastoreSegment)
	segment.End()
}
