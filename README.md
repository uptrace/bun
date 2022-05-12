# SQL-first Golang ORM for PostgreSQL, MySQL, MSSQL, and SQLite

[![build workflow](https://github.com/uptrace/bun/actions/workflows/build.yml/badge.svg)](https://github.com/uptrace/bun/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun)](https://pkg.go.dev/github.com/uptrace/bun)
[![Documentation](https://img.shields.io/badge/bun-documentation-informational)](https://bun.uptrace.dev/)
[![Chat](https://discordapp.com/api/guilds/752070105847955518/widget.png)](https://discord.gg/rWtp5Aj)

> Bun is brought to you by :star: [**uptrace/uptrace**](https://github.com/uptrace/uptrace). Uptrace
> is an open source and blazingly fast
> [distributed tracing tool](https://get.uptrace.dev/compare/distributed-tracing-tools.html) powered
> by OpenTelemetry and ClickHouse. Give it a star as well!

## Features

- Works with [PostgreSQL](https://bun.uptrace.dev/guide/drivers.html#postgresql),
  [MySQL](https://bun.uptrace.dev/guide/drivers.html#mysql) (including MariaDB),
  [MSSQL](https://bun.uptrace.dev/guide/drivers.html#mssql),
  [SQLite](https://bun.uptrace.dev/guide/drivers.html#sqlite).
- [ORM-like](/example/basic/) experience using good old SQL. Bun supports structs, map, scalars, and
  slices of map/structs/scalars.
- [Bulk inserts](https://bun.uptrace.dev/guide/query-insert.html).
- [Bulk updates](https://bun.uptrace.dev/guide/query-update.html) using common table expressions.
- [Bulk deletes](https://bun.uptrace.dev/guide/query-delete.html).
- [Fixtures](https://bun.uptrace.dev/guide/fixtures.html).
- [Migrations](https://bun.uptrace.dev/guide/migrations.html).
- [Soft deletes](https://bun.uptrace.dev/guide/soft-deletes.html).

Resources:

- [**Get started**](https://bun.uptrace.dev/guide/golang-orm.html)
- [Examples](https://github.com/uptrace/bun/tree/master/example)
- [Discussions](https://github.com/uptrace/bun/discussions)
- [Chat](https://discord.gg/rWtp5Aj)
- [Reference](https://pkg.go.dev/github.com/uptrace/bun)
- [Starter kit](https://github.com/go-bun/bun-starter-kit)

Projects using Bun:

- [gotosocial](https://github.com/superseriousbusiness/gotosocial) - Golang fediverse server.
- [alexedwards/scs](https://github.com/alexedwards/scs) - HTTP Session Management for Go.
- [emerald-web3-gateway](https://github.com/oasisprotocol/emerald-web3-gateway) - Web3 Gateway for
  the Oasis Emerald paratime.
- [lndhub.go](https://github.com/getAlby/lndhub.go) - accounting wrapper for the Lightning Network.
- [RealWorld app](https://github.com/go-bun/bun-realworld-app)
- And hundreds more.

## Benchmark

[https://github.com/davars/dbeval](https://github.com/davars/dbeval)

<details>
<summary>results</summary>

```
BenchmarkInsert
BenchmarkInsert/*dbeval.Memory/Authors
BenchmarkInsert/*dbeval.Memory/Authors-4         	   84450	     12104 ns/op	    2623 B/op	      70 allocs/op
BenchmarkInsert/*dbeval.Xorm/Authors
BenchmarkInsert/*dbeval.Xorm/Authors-4           	    7291	    153505 ns/op	    9024 B/op	     311 allocs/op
BenchmarkInsert/*dbeval.UpperDB/Authors
BenchmarkInsert/*dbeval.UpperDB/Authors-4        	    4608	    223672 ns/op	   24160 B/op	    1100 allocs/op
BenchmarkInsert/*dbeval.Bun/Authors
BenchmarkInsert/*dbeval.Bun/Authors-4            	    6034	    186439 ns/op	    6818 B/op	      80 allocs/op
BenchmarkInsert/*dbeval.PQ/Authors
BenchmarkInsert/*dbeval.PQ/Authors-4             	    1141	    907494 ns/op	    6487 B/op	     193 allocs/op
BenchmarkInsert/*dbeval.SQLX/Authors
BenchmarkInsert/*dbeval.SQLX/Authors-4           	    1165	    916987 ns/op	   10089 B/op	     271 allocs/op
BenchmarkInsert/*dbeval.Ozzo/Authors
BenchmarkInsert/*dbeval.Ozzo/Authors-4           	    1105	   1058082 ns/op	   27826 B/op	     588 allocs/op
BenchmarkInsert/*dbeval.PGXStdlib/Authors
BenchmarkInsert/*dbeval.PGXStdlib/Authors-4      	    1228	    900207 ns/op	    6032 B/op	     180 allocs/op
BenchmarkInsert/*dbeval.Gorm/Authors
BenchmarkInsert/*dbeval.Gorm/Authors-4           	     946	   1184285 ns/op	   35634 B/op	     918 allocs/op
BenchmarkInsert/*dbeval.PGX/Authors
BenchmarkInsert/*dbeval.PGX/Authors-4            	    1116	    923728 ns/op	    3839 B/op	     130 allocs/op
BenchmarkInsert/*dbeval.DBR/Authors
BenchmarkInsert/*dbeval.DBR/Authors-4            	    5800	    183982 ns/op	    8646 B/op	     230 allocs/op
BenchmarkInsert/*dbeval.GoPG/Authors
BenchmarkInsert/*dbeval.GoPG/Authors-4           	    6110	    173923 ns/op	    2906 B/op	      87 allocs/op

BenchmarkInsert/*dbeval.DBR/Articles
BenchmarkInsert/*dbeval.DBR/Articles-4           	    1706	    684466 ns/op	  133346 B/op	    1614 allocs/op
BenchmarkInsert/*dbeval.PQ/Articles
BenchmarkInsert/*dbeval.PQ/Articles-4            	     884	   1249791 ns/op	  100403 B/op	    1491 allocs/op
BenchmarkInsert/*dbeval.PGX/Articles
BenchmarkInsert/*dbeval.PGX/Articles-4           	     916	   1288143 ns/op	   83539 B/op	    1392 allocs/op
BenchmarkInsert/*dbeval.GoPG/Articles
BenchmarkInsert/*dbeval.GoPG/Articles-4          	    1726	    622639 ns/op	   78638 B/op	    1359 allocs/op
BenchmarkInsert/*dbeval.SQLX/Articles
BenchmarkInsert/*dbeval.SQLX/Articles-4          	     860	   1262599 ns/op	   92030 B/op	    1574 allocs/op
BenchmarkInsert/*dbeval.Gorm/Articles
BenchmarkInsert/*dbeval.Gorm/Articles-4          	     782	   1421550 ns/op	  136534 B/op	    2411 allocs/op
BenchmarkInsert/*dbeval.PGXStdlib/Articles
BenchmarkInsert/*dbeval.PGXStdlib/Articles-4     	     938	   1230576 ns/op	   86743 B/op	    1441 allocs/op
BenchmarkInsert/*dbeval.Bun/Articles
BenchmarkInsert/*dbeval.Bun/Articles-4           	    1843	    626681 ns/op	  101610 B/op	    1323 allocs/op
BenchmarkInsert/*dbeval.Xorm/Articles
BenchmarkInsert/*dbeval.Xorm/Articles-4          	    1677	    650244 ns/op	  126677 B/op	    1752 allocs/op
BenchmarkInsert/*dbeval.Memory/Articles
BenchmarkInsert/*dbeval.Memory/Articles-4        	    1988	   1223308 ns/op	   77576 B/op	    1310 allocs/op
BenchmarkInsert/*dbeval.UpperDB/Articles
BenchmarkInsert/*dbeval.UpperDB/Articles-4       	    1696	    687130 ns/op	  139680 B/op	    2862 allocs/op
BenchmarkInsert/*dbeval.Ozzo/Articles
BenchmarkInsert/*dbeval.Ozzo/Articles-4          	     697	   1496859 ns/op	  114780 B/op	    1950 allocs/op

BenchmarkFindAuthorByID
BenchmarkFindAuthorByID/*dbeval.UpperDB
BenchmarkFindAuthorByID/*dbeval.UpperDB-4        	   10184	    117527 ns/op	    9953 B/op	     441 allocs/op
BenchmarkFindAuthorByID/*dbeval.Bun
BenchmarkFindAuthorByID/*dbeval.Bun-4            	   20716	     54261 ns/op	    5096 B/op	      15 allocs/op
BenchmarkFindAuthorByID/*dbeval.Ozzo
BenchmarkFindAuthorByID/*dbeval.Ozzo-4           	   11166	     91043 ns/op	    3088 B/op	      64 allocs/op
BenchmarkFindAuthorByID/*dbeval.PQ
BenchmarkFindAuthorByID/*dbeval.PQ-4             	   13875	     86171 ns/op	     844 B/op	      24 allocs/op
BenchmarkFindAuthorByID/*dbeval.PGX
BenchmarkFindAuthorByID/*dbeval.PGX-4            	   13846	     79983 ns/op	     719 B/op	      15 allocs/op
BenchmarkFindAuthorByID/*dbeval.Memory
BenchmarkFindAuthorByID/*dbeval.Memory-4         	14113720	        82.33 ns/op	       0 B/op	       0 allocs/op
BenchmarkFindAuthorByID/*dbeval.Xorm
BenchmarkFindAuthorByID/*dbeval.Xorm-4           	   12027	     98519 ns/op	    3633 B/op	     106 allocs/op
BenchmarkFindAuthorByID/*dbeval.Gorm
BenchmarkFindAuthorByID/*dbeval.Gorm-4           	   11521	    102241 ns/op	    6592 B/op	     143 allocs/op
BenchmarkFindAuthorByID/*dbeval.PGXStdlib
BenchmarkFindAuthorByID/*dbeval.PGXStdlib-4      	   13933	     82626 ns/op	    1174 B/op	      28 allocs/op
BenchmarkFindAuthorByID/*dbeval.DBR
BenchmarkFindAuthorByID/*dbeval.DBR-4            	   21920	     51175 ns/op	    1756 B/op	      39 allocs/op
BenchmarkFindAuthorByID/*dbeval.SQLX
BenchmarkFindAuthorByID/*dbeval.SQLX-4           	   13603	     80788 ns/op	    1327 B/op	      32 allocs/op
BenchmarkFindAuthorByID/*dbeval.GoPG
BenchmarkFindAuthorByID/*dbeval.GoPG-4           	   23174	     50042 ns/op	     869 B/op	      17 allocs/op

BenchmarkFindAuthorByName
BenchmarkFindAuthorByName/*dbeval.SQLX
BenchmarkFindAuthorByName/*dbeval.SQLX-4         	    1070	   1065272 ns/op	  126348 B/op	    4018 allocs/op
BenchmarkFindAuthorByName/*dbeval.Bun
BenchmarkFindAuthorByName/*dbeval.Bun-4          	     877	   1231377 ns/op	  115803 B/op	    5005 allocs/op
BenchmarkFindAuthorByName/*dbeval.Xorm
BenchmarkFindAuthorByName/*dbeval.Xorm-4         	     471	   2345445 ns/op	  455711 B/op	   19080 allocs/op
BenchmarkFindAuthorByName/*dbeval.DBR
BenchmarkFindAuthorByName/*dbeval.DBR-4          	     954	   1089977 ns/op	  120070 B/op	    6023 allocs/op
BenchmarkFindAuthorByName/*dbeval.PQ
BenchmarkFindAuthorByName/*dbeval.PQ-4           	    1333	    784400 ns/op	   87159 B/op	    4006 allocs/op
BenchmarkFindAuthorByName/*dbeval.GoPG
BenchmarkFindAuthorByName/*dbeval.GoPG-4         	    1580	    770966 ns/op	   87525 B/op	    3028 allocs/op
BenchmarkFindAuthorByName/*dbeval.UpperDB
BenchmarkFindAuthorByName/*dbeval.UpperDB-4      	     789	   1314164 ns/op	  190689 B/op	    6428 allocs/op
BenchmarkFindAuthorByName/*dbeval.Ozzo
BenchmarkFindAuthorByName/*dbeval.Ozzo-4         	     948	   1255282 ns/op	  238764 B/op	    6053 allocs/op
BenchmarkFindAuthorByName/*dbeval.PGXStdlib
BenchmarkFindAuthorByName/*dbeval.PGXStdlib-4    	    1279	    920391 ns/op	  126163 B/op	    4014 allocs/op
BenchmarkFindAuthorByName/*dbeval.PGX
BenchmarkFindAuthorByName/*dbeval.PGX-4          	    1364	    780970 ns/op	  101967 B/op	    2028 allocs/op
BenchmarkFindAuthorByName/*dbeval.Gorm
BenchmarkFindAuthorByName/*dbeval.Gorm-4         	     340	   3445818 ns/op	 1573637 B/op	   27102 allocs/op
BenchmarkFindAuthorByName/*dbeval.Memory
BenchmarkFindAuthorByName/*dbeval.Memory-4       	38081223	        31.24 ns/op	       0 B/op	       0 allocs/op

BenchmarkRecentArticles
BenchmarkRecentArticles/*dbeval.PGXStdlib
BenchmarkRecentArticles/*dbeval.PGXStdlib-4      	     358	   3344119 ns/op	 3425578 B/op	   14177 allocs/op
BenchmarkRecentArticles/*dbeval.GoPG
BenchmarkRecentArticles/*dbeval.GoPG-4           	     364	   3156372 ns/op	 1794091 B/op	   10032 allocs/op
BenchmarkRecentArticles/*dbeval.Xorm
BenchmarkRecentArticles/*dbeval.Xorm-4           	     157	   7567835 ns/op	 5018011 B/op	   81425 allocs/op
BenchmarkRecentArticles/*dbeval.Gorm
BenchmarkRecentArticles/*dbeval.Gorm-4           	     139	   7980084 ns/op	 6776277 B/op	   85418 allocs/op
BenchmarkRecentArticles/*dbeval.SQLX
BenchmarkRecentArticles/*dbeval.SQLX-4           	     338	   3289802 ns/op	 3425890 B/op	   14181 allocs/op
BenchmarkRecentArticles/*dbeval.Ozzo
BenchmarkRecentArticles/*dbeval.Ozzo-4           	     320	   3508322 ns/op	 4025966 B/op	   18207 allocs/op
BenchmarkRecentArticles/*dbeval.DBR
BenchmarkRecentArticles/*dbeval.DBR-4            	     237	   5248644 ns/op	 3331003 B/op	   21370 allocs/op
BenchmarkRecentArticles/*dbeval.Bun
BenchmarkRecentArticles/*dbeval.Bun-4            	     280	   4528582 ns/op	 1864362 B/op	   15965 allocs/op
BenchmarkRecentArticles/*dbeval.UpperDB
BenchmarkRecentArticles/*dbeval.UpperDB-4        	     297	   3704663 ns/op	 3607287 B/op	   18542 allocs/op
BenchmarkRecentArticles/*dbeval.PQ
BenchmarkRecentArticles/*dbeval.PQ-4             	     308	   3489229 ns/op	 3277050 B/op	   17359 allocs/op
BenchmarkRecentArticles/*dbeval.Memory
BenchmarkRecentArticles/*dbeval.Memory-4         	29590380	        42.27 ns/op	       0 B/op	       0 allocs/op
BenchmarkRecentArticles/*dbeval.PGX
BenchmarkRecentArticles/*dbeval.PGX-4            	     356	   3345500 ns/op	 3297316 B/op	    6226 allocs/op
```

</details>

[https://github.com/frederikhors/orm-benchmark](https://github.com/frederikhors/orm-benchmark)

<details>
<summary>results</summary>

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

var items []map[string]interface{}
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

And scan results into scalars, structs, maps, slices of structs/maps/scalars:

```go
users := make([]User, 0)
if err := db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx); err != nil {
	panic(err)
}

user1 := new(User)
if err := db.NewSelect().Model(user1).Where("id = ?", 1).Scan(ctx); err != nil {
	panic(err)
}
```

See [**Getting started**](https://bun.uptrace.dev/guide/golang-orm.html) guide and check
[examples](example).

## See also

- [Golang HTTP router](https://github.com/uptrace/bunrouter)
- [Golang ClickHouse ORM](https://github.com/uptrace/go-clickhouse)
- [Golang msgpack](https://github.com/vmihailenco/msgpack)

## Contributors

Thanks to all the people who already contributed!

<a href="https://github.com/uptrace/bun/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=uptrace/bun" />
</a>
