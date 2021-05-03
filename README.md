# Tasty ORM for Go

bun is a DB-agnostic ORM for `*sql.DB`.

## Installation

```go
go get github.com/uptrace/bun
```

## Quickstart

First you need to create a `*sql.DB`. Here we using a SQLite3 driver.

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

db := bun.Open(sqldb, sqlitedialect.New())
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
	Limit(100).
	Scan(ctx)
```

The code above is equivalent to:

```go
query := "SELECT id, name FROM users AS user WHERE name != '' ORDER BY id ASC LIMIT 100"

rows, err := sqldb.QueryRows(ctx, query)
if err != nil {
	panic(err)
}

user := new(User)
if err := db.ScanRows(ctx, rows, user); err != nil {
	panic(err)
}
```

For more details, please check [basic](example/basic) example.
