module github.com/uptrace/bun/example/basic

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

require (
	github.com/go-mysql-org/go-mysql v1.1.2
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/uptrace/bun v0.0.0-00010101000000-000000000000
	github.com/uptrace/bun/dialect/sqlitedialect v0.0.0-00010101000000-000000000000
)
