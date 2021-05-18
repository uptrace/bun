# Tasty ORM for sql.DB

[![build workflow](https://github.com/uptrace/bun/actions/workflows/build.yml/badge.svg)](https://github.com/uptrace/bun/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun)](https://pkg.go.dev/github.com/uptrace/bun)
[![Documentation](https://img.shields.io/badge/bun-documentation-informational)](https://bun.uptrace.dev/)
[![Chat](https://discordapp.com/api/guilds/752070105847955518/widget.png)](https://discord.gg/rWtp5Aj)

## Installation

```go
go get github.com/uptrace/bun
```

## Quickstart

First you need to create a `*sql.DB`. Here we using the SQLite3 driver.

```go
import _ "github.com/mattn/go-sqlite3"

sqldb, err := sql.Open("sqlite3", ":memory:?cache=shared")
if err != nil {
	panic(err)
}
```

And then create a `*bun.DB` on top of it using the corresponding SQLite dialect:

```go
import (
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

db := bun.NewDB(sqldb, sqlitedialect.New())
```

Now you are ready to issue some queries:

```go
type User struct {
	ID	 int64
	Name string
}

user := new(User)
err := db.NewSelect().
	Model(user).
	Where("name != ?", "").
	OrderExpr("id ASC").
	Limit(1).
	Scan(ctx)
```

The code above is equivalent to:

```go
query := "SELECT id, name FROM users AS user WHERE name != '' ORDER BY id ASC LIMIT 1"

rows, err := sqldb.QueryContext(ctx, query)
if err != nil {
	panic(err)
}

if rows.Next() {
	user := new(User)
	if err := db.ScanRow(ctx, rows, user); err != nil {
		panic(err)
	}
}

if err := rows.Err(); err != nil {
    panic(err)
}
```

For more details, please check [docs](https://bun.uptrace.dev/).
