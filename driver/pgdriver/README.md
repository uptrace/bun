# pgdriver

[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun/driver/pgdriver)](https://pkg.go.dev/github.com/uptrace/bun/driver/pgdriver)

pgdriver is a database/sql driver for PostgreSQL based on [go-pg](https://github.com/go-pg/pg) code.

You can install it with:

```shell
go get github.com/uptrace/bun/driver/pgdriver
```

And then create a `sql.DB` using it:

```go
import _ "github.com/uptrace/bun/driver/pgdriver"

dsn := "postgres://postgres:@localhost:5432/test"
db, err := sql.Open("pg", dsn)
```

Alternatively:

```go
dsn := "postgres://postgres:@localhost:5432/test"
db := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
```

[Benchmark](https://github.com/go-bun/bun-benchmark):

```
BenchmarkInsert/pg-12 	            7254	    148380 ns/op	     900 B/op	      13 allocs/op
BenchmarkInsert/pgx-12         	    6494	    166391 ns/op	    2076 B/op	      26 allocs/op
BenchmarkSelect/pg-12          	    9100	    132952 ns/op	    1417 B/op	      18 allocs/op
BenchmarkSelect/pgx-12         	    8199	    154920 ns/op	    3679 B/op	      60 allocs/op
```

See [documentation](https://bun.uptrace.dev/postgres/) for more details.
