package bundebug

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/fatih/color"

	"github.com/uptrace/bun"
)

type ConfigOption func(*QueryHook)

func WithVerbose(on bool) ConfigOption {
	return func(h *QueryHook) {
		h.verbose = on
	}
}

type QueryHook struct {
	verbose bool
}

var _ bun.QueryHook = (*QueryHook)(nil)

func NewQueryHook(opts ...ConfigOption) *QueryHook {
	h := new(QueryHook)
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *QueryHook) BeforeQuery(
	ctx context.Context, event *bun.QueryEvent,
) context.Context {
	return ctx
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if !h.verbose {
		switch event.Err {
		case nil, sql.ErrNoRows:
			return
		}
	}

	now := time.Now()
	dur := now.Sub(event.StartTime)

	args := []interface{}{
		"[bun]",
		now.Format(" 15:04:05.000 "),
		formatOperation(event),
		fmt.Sprintf(" %10s ", dur.Round(time.Microsecond)),
		event.Query,
	}

	if event.Err != nil {
		typ := reflect.TypeOf(event.Err).String()
		args = append(args,
			"\t",
			color.New(color.BgRed).Sprintf(" %s ", typ+": "+event.Err.Error()),
		)
	}

	fmt.Println(args...)
}

func formatOperation(event *bun.QueryEvent) string {
	operation := event.Operation()
	return operationColor(operation).Sprintf(" %-16s ", operation)
}

func operationColor(operation string) *color.Color {
	switch operation {
	case "SELECT":
		return color.New(color.BgGreen, color.FgHiWhite)
	case "INSERT":
		return color.New(color.BgBlue, color.FgHiWhite)
	case "UPDATE":
		return color.New(color.BgYellow, color.FgHiBlack)
	case "DELETE":
		return color.New(color.BgMagenta, color.FgHiWhite)
	default:
		return color.New(color.BgWhite, color.FgHiBlack)
	}
}
