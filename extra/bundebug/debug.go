package bundebug

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/uptrace/bun"
)

type ConfigOption func(*QueryHook)

func WithVerbose() ConfigOption {
	return func(h *QueryHook) {
		h.verbose = true
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

func eventOperation(event *bun.QueryEvent) string {
	if event.QueryAppender != nil {
		return event.QueryAppender.Operation()
	}
	return queryOperation(event.Query)
}

func queryOperation(name string) string {
	if idx := strings.IndexByte(name, ' '); idx > 0 {
		name = name[:idx]
	}
	if len(name) > 16 {
		name = name[:16]
	}
	return name
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
