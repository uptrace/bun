package bunzerolog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/uptrace/bun"
)

type Record struct {
	Level     zerolog.Level `json:"level"`
	Error     string        `json:"error"`
	Status    string        `json:"status"`
	Query     string        `json:"query"`
	Operation string        `json:"operation"`
	Duration  string        `json:"duration"`
}

func TestAfterQuery(t *testing.T) {
	testCases := []struct {
		ctx                context.Context
		name               string
		queryLogLevel      zerolog.Level
		errorQueryLogLevel zerolog.Level
		slowQueryLogLevel  zerolog.Level
		slowQueryThreshold time.Duration
		event              *bun.QueryEvent
		now                func() time.Time
		expect             Record
	}{
		{
			ctx:           context.Background(),
			name:          "basic query log",
			queryLogLevel: zerolog.DebugLevel,
			event: &bun.QueryEvent{
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				StartTime: time.Date(2006, 1, 2, 15, 4, 2, 0, time.Local),
				Err:       nil,
			},
			now: func() time.Time { return time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local) },
			expect: Record{
				Level:     zerolog.DebugLevel,
				Error:     "",
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				Operation: "SELECT",
				Duration:  "3s",
			},
		},
		{
			ctx:                context.Background(),
			name:               "does not become slow query when below slowQueryThreshold",
			queryLogLevel:      zerolog.DebugLevel,
			slowQueryLogLevel:  zerolog.WarnLevel,
			slowQueryThreshold: 3 * time.Second,
			event: &bun.QueryEvent{
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				StartTime: time.Date(2006, 1, 2, 15, 4, 3, 0, time.Local),
				Err:       nil,
			},
			now: func() time.Time { return time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local) },
			expect: Record{
				Level:     zerolog.DebugLevel,
				Error:     "",
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				Operation: "SELECT",
				Duration:  "2s",
			},
		},
		{
			ctx:                context.Background(),
			name:               "becomes slow query when at or above slowQueryThreshold",
			slowQueryLogLevel:  zerolog.WarnLevel,
			slowQueryThreshold: 3 * time.Second,
			event: &bun.QueryEvent{
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				StartTime: time.Date(2006, 1, 2, 15, 4, 2, 0, time.Local),
				Err:       nil,
			},
			now: func() time.Time { return time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local) },
			expect: Record{
				Level:     zerolog.WarnLevel,
				Error:     "",
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				Operation: "SELECT",
				Duration:  "3s",
			},
		},
		{
			ctx:                context.Background(),
			name:               "error query log",
			errorQueryLogLevel: zerolog.ErrorLevel,
			event: &bun.QueryEvent{
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				Err:       errors.New("unexpected error"),
				StartTime: time.Date(2006, 1, 2, 15, 4, 2, 0, time.Local),
			},
			now: func() time.Time { return time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local) },
			expect: Record{
				Level:     zerolog.ErrorLevel,
				Error:     "unexpected error",
				Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
				Operation: "SELECT",
				Duration:  "3s",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"(with global logger)", func(t *testing.T) {
			var buf bytes.Buffer
			ctx := zerolog.New(&buf).Level(tc.expect.Level).WithContext(tc.ctx)

			hook := NewQueryHook(
				WithQueryLogLevel(tc.queryLogLevel),
				WithErrorQueryLogLevel(tc.errorQueryLogLevel),
				WithSlowQueryLogLevel(tc.slowQueryLogLevel),
				WithSlowQueryThreshold(tc.slowQueryThreshold),
			)
			hook.now = tc.now
			hook.AfterQuery(ctx, tc.event)

			var result Record
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			if !reflect.DeepEqual(tc.expect, result) {
				t.Errorf("unexpected logging want=%+v but got=%+v", tc.expect, result)
			}
		})

		t.Run(tc.name+"(with logger instance)", func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf).Level(tc.expect.Level)

			hook := NewQueryHook(
				WithQueryLogLevel(tc.queryLogLevel),
				WithErrorQueryLogLevel(tc.errorQueryLogLevel),
				WithSlowQueryLogLevel(tc.slowQueryLogLevel),
				WithSlowQueryThreshold(tc.slowQueryThreshold),
				WithLogger(&logger),
			)
			hook.now = tc.now
			hook.AfterQuery(tc.ctx, tc.event)

			var result Record
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			if !reflect.DeepEqual(tc.expect, result) {
				t.Errorf("unexpected logging want=%+v but got=%+v", tc.expect, result)
			}
		})
	}

	t.Run("custom format", func(t *testing.T) {
		expect := struct {
			Level zerolog.Level
			Query string
		}{
			Level: zerolog.DebugLevel,
			Query: "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
		}
		var buf bytes.Buffer
		ctx := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(context.Background())

		hook := NewQueryHook(
			WithLogFormat(func(ctx context.Context, event *bun.QueryEvent, zeroevent *zerolog.Event) *zerolog.Event {
				return zeroevent.Str("query", event.Query)
			}),
		)
		hook.now = func() time.Time { return time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local) }
		event := &bun.QueryEvent{
			Query:     "SELECT `user`.`id`, `user`.`name`, `user`.`email` FROM `users`",
			Err:       nil,
			StartTime: time.Date(2006, 1, 2, 15, 4, 2, 0, time.Local),
		}
		hook.AfterQuery(ctx, event)

		var result struct {
			Level zerolog.Level
			Query string
		}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if !reflect.DeepEqual(expect, result) {
			t.Errorf("unexpected logging want=%+v but got=%+v", expect, result)
		}
	})
}
