# Simple and performant ORM for sql.DB

[![build workflow](https://github.com/uptrace/bun/actions/workflows/build.yml/badge.svg)](https://github.com/uptrace/bun/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun)](https://pkg.go.dev/github.com/uptrace/bun)
[![Documentation](https://img.shields.io/badge/bun-documentation-informational)](https://bun.uptrace.dev/)
[![Chat](https://discordapp.com/api/guilds/752070105847955518/widget.png)](https://discord.gg/rWtp5Aj)

Main features are:

- Works with [PostgreSQL](https://bun.uptrace.dev/guide/drivers.html#postgresql),
  [MySQL](https://bun.uptrace.dev/guide/drivers.html#mysql),
  [SQLite](https://bun.uptrace.dev/guide/drivers.html#sqlite).
- [Selecting](/example/basic/) into a map, struct, slice of maps/structs/vars.
- [Bulk inserts](https://bun.uptrace.dev/guide/queries.html#insert).
- [Bulk updates](https://bun.uptrace.dev/guide/queries.html#update) using common table expression.
- [Bulk deletes](https://bun.uptrace.dev/guide/queries.html#delete).
- [Fixtures](https://bun.uptrace.dev/guide/fixtures.html).
- [Migrations](https://bun.uptrace.dev/guide/migrations.html).

Resources:

- [Examples](https://github.com/uptrace/bun/tree/master/example)
- [Documentation](https://bun.uptrace.dev/)
- [Reference](https://pkg.go.dev/github.com/uptrace/bun)
- [Starter kit](https://github.com/go-bun/bun-starter-kit)
- [RealWorld app](https://github.com/go-bun/bun-realworld-app)

## Installation

```go
go get github.com/uptrace/bun
```

## Quickstart

First you need to create a `sql.DB`. Here we using the SQLite3 driver.

```go
import _ "github.com/mattn/go-sqlite3"

sqldb, err := sql.Open("sqlite3", ":memory:?cache=shared")
if err != nil {
	panic(err)
}
```

And then create a `bun.DB` on top of it using the corresponding SQLite dialect:

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

if !rows.Next() {
    panic(sql.ErrNoRows)
}

user := new(User)
if err := db.ScanRow(ctx, rows, user); err != nil {
	panic(err)
}

if err := rows.Err(); err != nil {
    panic(err)
}
```

## Basic example

For our [basic example](/example/basic/) we need to load some data first. To provide initial data,
use Bun [fixtures](https://bun.uptrace.dev/guide/fixtures.html):

```go
import "github.com/uptrace/bun/dbfixture"

// Register models for the fixture.
db.RegisterModel((*User)(nil), (*Story)(nil))

// WithRecreateTables tells Bun to drop existing tables and create new ones.
fixture := dbfixture.New(db, dbfixture.WithRecreateTables())

// Load fixture.yaml which contains data for User and Story models.
if err := fixture.Load(ctx, os.DirFS("."), "fixture.yaml"); err != nil {
	panic(err)
}
```

The `fixture.yaml` looks like this:

```yaml
- model: User
  rows:
    - _id: admin
      name: admin
      emails: ['admin1@admin', 'admin2@admin']
    - _id: root
      name: root
      emails: ['root1@root', 'root2@root']

- model: Story
  rows:
    - title: Cool story
      author_id: '{{ $.User.admin.ID }}'
```

To select all users:

```go
users := make([]User, 0)
if err := db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx); err != nil {
	panic(err)
}
```

To select a single user by id:

```go
user1 := new(User)
if err := db.NewSelect().Model(user1).Where("id = ?", 1).Scan(ctx); err != nil {
	panic(err)
}
```

To select a story and the associated author in a single query:

```go
story := new(Story)
if err := db.NewSelect().
	Model(story).
	Relation("Author").
	Limit(1).
	Scan(ctx); err != nil {
	panic(err)
}
```

To select a user into a map:

```go
m := make(map[string]interface{})
if err := db.NewSelect().
	Model((*User)(nil)).
	Limit(1).
	Scan(ctx, &m); err != nil {
	panic(err)
}
```

To select all users scanning each column into a separate slice:

```go
var ids []int64
var names []string
if err := db.NewSelect().
	ColumnExpr("id, name").
	Model((*User)(nil)).
	OrderExpr("id ASC").
	Scan(ctx, &ids, &names); err != nil {
	panic(err)
}
```

For more details, please consult [docs](https://bun.uptrace.dev/) and check [examples](example).
