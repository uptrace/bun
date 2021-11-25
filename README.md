<p align="center">
  <a href="https://uptrace.dev/?utm_source=gh-redis&utm_campaign=gh-redis-banner1">
    <img src="https://raw.githubusercontent.com/uptrace/roadmap/master/banner1.png" alt="All-in-one tool to optimize performance and monitor errors & logs">
  </a>
</p>

# Simple and performant client for PostgreSQL, MySQL, and SQLite

[![build workflow](https://github.com/uptrace/bun/actions/workflows/build.yml/badge.svg)](https://github.com/uptrace/bun/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun)](https://pkg.go.dev/github.com/uptrace/bun)
[![Documentation](https://img.shields.io/badge/bun-documentation-informational)](https://bun.uptrace.dev/)

**Status**: API freeze (stable release). Note that all sub-packages (mainly extra/\* packages) are
not part of the API freeze and are developed independently. You can think of them as of 3rd party
packages that share one repo with the core.

Main features are:

- Works with [PostgreSQL](https://bun.uptrace.dev/guide/drivers.html#postgresql),
  [MySQL](https://bun.uptrace.dev/guide/drivers.html#mysql) (including MariaDB),
  [SQLite](https://bun.uptrace.dev/guide/drivers.html#sqlite).
- [Selecting](/example/basic/) into scalars, structs, maps, slices of maps/structs/scalars.
- [Bulk inserts](https://bun.uptrace.dev/guide/query-insert.html).
- [Bulk updates](https://bun.uptrace.dev/guide/query-update.html) using common table expressions.
- [Bulk deletes](https://bun.uptrace.dev/guide/query-delete.html).
- [Fixtures](https://bun.uptrace.dev/guide/fixtures.html).
- [Migrations](https://bun.uptrace.dev/guide/migrations.html).
- [Soft deletes](https://bun.uptrace.dev/guide/soft-deletes.html).

Resources:

- [**Get started**](https://bun.uptrace.dev/guide/getting-started.html)
- [Discussions](https://github.com/uptrace/bun/discussions)
- [Examples](https://github.com/uptrace/bun/tree/master/example)
- [Newsletter](https://blog.uptrace.dev/pages/newsletter.html) to get latest updates.
- [Reference](https://pkg.go.dev/github.com/uptrace/bun)
- [Starter kit](https://github.com/go-bun/bun-starter-kit)

Projects using Bun:

- [gotosocial](https://github.com/superseriousbusiness/gotosocial) - Golang fediverse server.
- [qvalet](https://github.com/cmaster11/qvalet) listens for HTTP requests and executes commands on
  demand.
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

## Why another database client?

So you can elegantly write complex queries:

```go
regionalSales := db.NewSelect().
	ColumnExpr("region").
	ColumnExpr("SUM(amount) AS total_sales").
	TableExpr("orders").
	GroupExpr("region")

topRegions := db.NewSelect().
	ColumnExpr("region").
	TableExpr("regional_sales").
	Where("total_sales > (SELECT SUM(total_sales) / 10 FROM regional_sales)")

var items map[string]interface{}
err := db.NewSelect().
	With("regional_sales", regionalSales).
	With("top_regions", topRegions).
	ColumnExpr("region").
	ColumnExpr("product").
	ColumnExpr("SUM(quantity) AS product_units").
	ColumnExpr("SUM(amount) AS product_sales").
	TableExpr("orders").
	Where("region IN (SELECT region FROM top_regions)").
	GroupExpr("region").
	GroupExpr("product").
	Scan(ctx, &items)
```

```sql
WITH regional_sales AS (
    SELECT region, SUM(amount) AS total_sales
    FROM orders
    GROUP BY region
), top_regions AS (
    SELECT region
    FROM regional_sales
    WHERE total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)
)
SELECT region,
       product,
       SUM(quantity) AS product_units,
       SUM(amount) AS product_sales
FROM orders
WHERE region IN (SELECT region FROM top_regions)
GROUP BY region, product
```

And scan results into scalars, structs, maps, slices of structs/maps/scalars.

## Installation

```go
go get github.com/uptrace/bun
```

You also need to install a database/sql driver and the corresponding Bun
[dialect](https://bun.uptrace.dev/guide/drivers.html).

See [**Getting started**](https://bun.uptrace.dev/guide/getting-started.html) guide and check
[examples](example).

## Contributors

Thanks to all the people who already contributed!

<a href="https://github.com/uptrace/bun/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=uptrace/bun" />
</a>
