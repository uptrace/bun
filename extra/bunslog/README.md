# bunslog

bunslog is a logging package for Bun that uses slog.  
This package enables SQL queries executed by Bun to be logged and displayed using slog.

## Installation

```bash
go get github.com/uptrace/bun/extra/bunslog
```

## Features

- Supports setting a `*slog.Logger` instance or uses the global logger if not set.
- Logs general SQL queries with configurable log levels.
- Logs slow SQL queries based on a configurable duration threshold.
- Logs SQL queries that result in errors, for easier debugging.
- Allows for custom log formatting.

## Usage

First, import the bunslog package:
```go
import "github.com/uptrace/bun/extra/bunslog"
```

Then, create a new QueryHook and add the hook to `*bun.DB` instance:
```go
db := bun.NewDB(sqldb, dialect)

hook := bunslog.NewQueryHook(
	bunslog.WithQueryLogLevel(slog.LevelDebug),
	bunslog.WithSlowQueryLogLevel(slog.LevelWarn),
	bunslog.WithErrorQueryLogLevel(slog.LevelError),
	bunslog.WithSlowQueryThreshold(3 * time.Second),
)

db.AddQueryHook(hook)
```

## Setting a Custom `*slog.Logger` Instance

To set a `*slog.Logger` instance, you can use the WithLogger option:

```go
logger := slog.NewLogger()
hook := bunslog.NewQueryHook(
	bunslog.WithLogger(logger),
	// other options...
)
```

If a `*slog.Logger` instance is not set, the global logger will be used.

## Custom Log Formatting

To customize the log format, you can use the WithLogFormat option:

```go
customFormat := func(event *bun.QueryEvent) []slog.Attr {
	// your custom formatting logic here
}

hook := bunslog.NewQueryHook(
	bunslog.WithLogFormat(customFormat),
	// other options...
)
```

## Options

- `WithLogger(logger *slog.Logger)`: Sets a `*slog.Logger` instance. If not set, the global logger will be used.
- `WithQueryLogLevel(level slog.Level)`: Sets the log level for general queries.
- `WithSlowQueryLogLevel(level slog.Level)`: Sets the log level for slow queries.
- `WithErrorQueryLogLevel(level slog.Level)`: Sets the log level for queries that result in errors.
- `WithSlowQueryThreshold(threshold time.Duration)`: Sets the duration threshold for identifying slow queries.
- `WithLogFormat(f logFormat)`: Sets the custom format for slog output.
