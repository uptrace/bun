package bundebug

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
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
	if !h.verbose && event.Err == nil {
		return
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
	operation := eventOperation(event)
	return operationColor(operation).Sprintf(" %-16s ", operation)
}

func eventOperation(event *bun.QueryEvent) string {
	switch event.QueryAppender.(type) {
	case *bun.SelectQuery:
		return "SELECT"
	case *bun.InsertQuery:
		return "INSERT"
	case *bun.UpdateQuery:
		return "UPDATE"
	case *bun.DeleteQuery:
		return "DELETE"
	case *bun.CreateTableQuery:
		return "CREATE TABLE"
	case *bun.DropTableQuery:
		return "DROP TABLE"
	}
	return queryOperation(event.Query)
}

func queryOperation(name []byte) string {
	if idx := bytes.IndexByte(name, ' '); idx > 0 {
		name = name[:idx]
	}
	if len(name) > 16 {
		name = name[:16]
	}
	return string(name)
}

func operationColor(operation string) *color.Color {
	switch operation {
	case "SELECT":
		return color.New(color.BgGreen)
	case "INSERT":
		return color.New(color.BgBlue)
	case "UPDATE":
		return color.New(color.BgYellow)
	case "DELETE":
		return color.New(color.BgRed)
	default:
		return color.New(color.FgBlack, color.BgWhite)
	}
}
