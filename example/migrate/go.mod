module github.com/uptrace/bun/example/migrate

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

require (
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/uptrace/bun v0.0.0-00010101000000-000000000000
	github.com/uptrace/bun/dialect/sqlitedialect v0.0.0-00010101000000-000000000000
	github.com/urfave/cli/v2 v2.3.0
)
