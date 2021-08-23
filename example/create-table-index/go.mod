module github.com/uptrace/bun/example/create-table-index

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

require (
	github.com/uptrace/bun v1.0.0-rc.1
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.0-rc.1
	github.com/uptrace/bun/driver/sqliteshim v1.0.0-rc.1
	github.com/uptrace/bun/extra/bundebug v1.0.0-rc.1
)
