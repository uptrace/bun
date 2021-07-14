<p align="center">
  <a href="https://uptrace.dev/?utm_source=gh-redis&utm_campaign=gh-redis-banner1">
    <img src="https://raw.githubusercontent.com/uptrace/roadmap/master/banner1.png" alt="All-in-one tool to optimize performance and monitor errors & logs">
  </a>
</p>

# Simple and performant SQL database client

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
- [Soft deletes](https://bun.uptrace.dev/guide/soft-deletes.html).

Resources:

- [Examples](https://github.com/uptrace/bun/tree/master/example)
- [Documentation](https://bun.uptrace.dev/)
- [Reference](https://pkg.go.dev/github.com/uptrace/bun)
- [Starter kit](https://github.com/go-bun/bun-starter-kit)
- [RealWorld app](https://github.com/go-bun/bun-realworld-app)

<details>
  <summary>github.com/frederikhors/orm-benchmark results</summary>

```
  4000 times - Insert
  raw_stmt:     0.38s        94280 ns/op     718 B/op     14 allocs/op
       raw:     0.39s        96719 ns/op     718 B/op     13 allocs/op
 beego_orm:     0.48s       118994 ns/op    2411 B/op     56 allocs/op
       bun:     0.57s       142285 ns/op     918 B/op     12 allocs/op
        pg:     0.58s       145496 ns/op    1235 B/op     12 allocs/op
      gorm:     0.70s       175294 ns/op    6665 B/op     88 allocs/op
      xorm:     0.76s       189533 ns/op    3032 B/op     94 allocs/op

  4000 times - MultiInsert 100 row
       raw:     4.59s      1147385 ns/op  135155 B/op    916 allocs/op
  raw_stmt:     4.59s      1148137 ns/op  131076 B/op    916 allocs/op
 beego_orm:     5.50s      1375637 ns/op  179962 B/op   2747 allocs/op
       bun:     6.18s      1544648 ns/op    4265 B/op    214 allocs/op
        pg:     7.01s      1753495 ns/op    5039 B/op    114 allocs/op
      gorm:     9.52s      2379219 ns/op  293956 B/op   3729 allocs/op
      xorm:    11.66s      2915478 ns/op  286140 B/op   7422 allocs/op

  4000 times - Update
  raw_stmt:     0.26s        65781 ns/op     773 B/op     14 allocs/op
       raw:     0.31s        77209 ns/op     757 B/op     13 allocs/op
 beego_orm:     0.43s       107064 ns/op    1802 B/op     47 allocs/op
       bun:     0.56s       139839 ns/op     589 B/op      4 allocs/op
        pg:     0.60s       149608 ns/op     896 B/op     11 allocs/op
      gorm:     0.74s       185970 ns/op    6604 B/op     81 allocs/op
      xorm:     0.81s       203240 ns/op    2994 B/op    119 allocs/op

  4000 times - Read
       raw:     0.33s        81671 ns/op    2081 B/op     49 allocs/op
  raw_stmt:     0.34s        85847 ns/op    2112 B/op     50 allocs/op
 beego_orm:     0.38s        94777 ns/op    2106 B/op     75 allocs/op
        pg:     0.42s       106148 ns/op    1526 B/op     22 allocs/op
       bun:     0.43s       106904 ns/op    1319 B/op     18 allocs/op
      gorm:     0.65s       162221 ns/op    5240 B/op    108 allocs/op
      xorm:     1.13s       281738 ns/op    8326 B/op    237 allocs/op

  4000 times - MultiRead limit 100
       raw:     1.52s       380351 ns/op   38356 B/op   1037 allocs/op
  raw_stmt:     1.54s       385541 ns/op   38388 B/op   1038 allocs/op
        pg:     1.86s       465468 ns/op   24045 B/op    631 allocs/op
       bun:     2.58s       645354 ns/op   30009 B/op   1122 allocs/op
 beego_orm:     2.93s       732028 ns/op   55280 B/op   3077 allocs/op
      gorm:     4.97s      1241831 ns/op   71628 B/op   3877 allocs/op
      xorm:     doesn't work
```

</details>

## Installation

```go
go get github.com/uptrace/bun
```

You also need to install a database/sql driver and the corresponding Bun
[dialect](https://bun.uptrace.dev/guide/drivers.html).

## Quickstart

First you need to create a `sql.DB`. Here we are using the
[sqliteshim](https://pkg.go.dev/github.com/uptrace/bun/driver/sqliteshim) driver which choses
between [modernc.org/sqlite](https://modernc.org/sqlite/) and
[mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) depending on your platform.

```go
import "github.com/uptrace/bun/driver/sqliteshim"

sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
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
	ID   int64
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

To provide initial data for our [example](/example/basic/), we will use Bun
[fixtures](https://bun.uptrace.dev/guide/fixtures.html):

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

## Contributors

Thanks to all the people who already contributed!

<a href="https://github.com/uptrace/bun/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=uptrace/bun" />
</a>
