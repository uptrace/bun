# sqliteshim

[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun/driver/sqliteshim)](https://pkg.go.dev/github.com/uptrace/bun/driver/sqliteshim)

sqliteshim driver choses between [modernc.org/sqlite](https://modernc.org/sqlite/) and
[mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) depending on your platform.

You can install it with:

```shell
github.com/uptrace/bun/driver/sqliteshim
```

And then create a `sql.DB` using it:

```go
sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
```
