package bundebug

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

type config struct {
	verbose bool
}

type ConfigOption func(*config)

func Verbose() ConfigOption {
	return func(cfg *config) {
		cfg.verbose = true
	}
}

type QueryHook struct {
	cfg config
}

var _ bun.QueryHook = (*QueryHook)(nil)

func NewQueryHook(opts ...ConfigOption) *QueryHook {
	h := new(QueryHook)
	for _, opt := range opts {
		opt(&h.cfg)
	}
	return h
}

func (h *QueryHook) BeforeQuery(
	ctx context.Context, event *bun.QueryEvent,
) context.Context {
	if event.Err != nil {
		fmt.Printf("%s executing a query:\n%s\n", event.Err, event.Query)
	} else if h.cfg.verbose {
		fmt.Println(event.Query)
	}

	return ctx
}

func (QueryHook) AfterQuery(context.Context, *bun.QueryEvent) {}
