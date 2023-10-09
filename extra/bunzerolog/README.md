# bunzerolog

bunzerolog is a logging package for Bun that uses zerolog.  
This package enables SQL queries executed by Bun to be logged and displayed using zerolog.

## Installation

```bash
go get github.com/uptrace/bun/extra/bunzerolog
```

## Features

- Supports setting a `*zerolog.Logger` instance or uses the global logger if not set.
- Supports setting a `*zerolog.Logger` instance using the context.
- Logs general SQL queries with configurable log levels.
- Logs slow SQL queries based on a configurable duration threshold.
- Logs SQL queries that result in errors, for easier debugging.
- Allows for custom log formatting.

## Usage

First, import the bunzerolog package:
```go
import "github.com/uptrace/bun/extra/bunzerolog"
```

Then, create a new QueryHook and add the hook to `*bun.DB` instance:
```go
import "github.com/rs/zerolog"

db := bun.NewDB(sqldb, dialect)

hook := bunzerolog.NewQueryHook(
    bunzerolog.WithQueryLogLevel(zerolog.DebugLevel),
    bunzerolog.WithSlowQueryLogLevel(zerolog.WarnLevel),
    bunzerolog.WithErrorQueryLogLevel(zerolog.ErrorLevel),
    bunzerolog.WithSlowQueryThreshold(3 * time.Second),
)

db.AddQueryHook(hook)
```

## Setting a Custom `*zerolog.Logger` Instance

To set a `*zerolog.Logger` instance, you can use the WithLogger option:

```go
logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
hook := bunzerolog.NewQueryHook(
    bunzerolog.WithLogger(logger),
	// other options...
)
```

If a `*zerolog.Logger` instance is not set, the logger from the context will be used.

## Custom Log Formatting

To customize the log format, you can use the WithLogFormat option:

```go
customFormat := func(ctx context.Context, event *bun.QueryEvent, zerevent *zerolog.Event) *zerolog.Event {
    duration := h.now().Sub(event.StartTime)
    
    return zerevent.
        Err(event.Err).
        Str("request_id", requestid.FromContext(ctx)).
        Str("query", event.Query).
        Str("operation", event.Operation()).
        Str("duration", duration.String())
}

hook := bunzerolog.NewQueryHook(
    bunzerolog.WithLogFormat(customFormat),
	// other options...
)
```

## Options

- `WithLogger(logger *zerolog.Logger)`: Sets a `*zerolog.Logger` instance. If not set, the logger from context will be used.
- `WithQueryLogLevel(level zerolog.Level)`: Sets the log level for general queries.
- `WithSlowQueryLogLevel(level zerolog.Level)`: Sets the log level for slow queries.
- `WithErrorQueryLogLevel(level zerolog.Level)`: Sets the log level for queries that result in errors.
- `WithSlowQueryThreshold(threshold time.Duration)`: Sets the duration threshold for identifying slow queries.
- `WithLogFormat(f logFormat)`: Sets the custom format for slog output.
