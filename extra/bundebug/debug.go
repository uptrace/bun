package bundebug

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/fatih/color"

	"github.com/uptrace/bun"
)

type ConfigOption func(*QueryHook)

// WithEnabled enables/disables the hook.
func WithEnabled(on bool) ConfigOption {
	return func(h *QueryHook) {
		h.enabled = on
	}
}

// WithVerbose configures the hook to log all queries
// (by default, only failed queries are logged).
func WithVerbose(on bool) ConfigOption {
	return func(h *QueryHook) {
		h.verbose = on
	}
}

// WithEnv configures the hook using then environment variable value.
// For example, WithEnv("BUNDEBUG"):
//    - BUNDEBUG=0 - disables the hook.
//    - BUNDEBUG=1 - enables the hook.
//    - BUNDEBUG=2 - enables the hook and verbose mode.
func FromEnv(key string) ConfigOption {
	if key == "" {
		key = "BUNDEBUG"
	}
	return func(h *QueryHook) {
		if env, ok := os.LookupEnv(key); ok {
			h.enabled = env != "" && env != "0"
			h.verbose = env == "2"
		}
	}
}

type QueryHook struct {
	enabled bool
	verbose bool
}

var _ bun.QueryHook = (*QueryHook)(nil)

func NewQueryHook(opts ...ConfigOption) *QueryHook {
	h := &QueryHook{
		enabled: true,
	}
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
	if !h.enabled {
		return
	}

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
