package bun

import (
	"context"
)

type contextKey struct{}

type scope struct {
	query Query
	model Model
	event *QueryEvent
}

var scopeKey = contextKey{}

func (s *scope) copy() *scope {
	return &scope{
		query: s.query,
		model: s.model,
		event: s.event,
	}
}

func scopeFromContext(ctx context.Context) *scope {
	state := ctx.Value(scopeKey)
	if state == nil {
		return &scope{}
	}
	return state.(*scope)
}

func withQueryScope(ctx context.Context, q Query) context.Context {
	scopeVal := scopeFromContext(ctx).copy()
	scopeVal.query = q
	return context.WithValue(ctx, scopeKey, scopeVal)
}

func withModel(ctx context.Context, m Model) context.Context {
	scopeVal := scopeFromContext(ctx).copy()
	scopeVal.model = m
	return context.WithValue(ctx, scopeKey, scopeVal)
}

func withEvent(ctx context.Context, e *QueryEvent) context.Context {
	scopeVal := scopeFromContext(ctx)
	scopeVal.event = e
	return context.WithValue(ctx, scopeKey, scopeVal.copy())
}

func isManagedQuery(ctx context.Context) bool {
	return scopeFromContext(ctx).query != nil
}
