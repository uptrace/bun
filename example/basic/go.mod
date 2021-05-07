module github.com/uptrace/bun/example/basic

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

require (
	github.com/go-mysql-org/go-mysql v1.1.2
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/pingcap/errors v0.11.4 // indirect
	github.com/siddontang/go-log v0.0.0-20190221022429-1e957dd83bed // indirect
	github.com/uptrace/bun v0.0.0-20210507075305-2e91d2c5c8de
	github.com/uptrace/bun/dialect/sqlitedialect v0.0.0-20210507070510-0d95488a5553
)
