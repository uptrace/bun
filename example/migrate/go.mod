module github.com/uptrace/bun/example/migrate

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/uptrace/bun v0.1.1
	github.com/uptrace/bun/dialect/sqlitedialect v0.0.0-20210507070510-0d95488a5553
	github.com/uptrace/bun/driver/sqliteshim v0.2.14
	github.com/uptrace/bun/extra/bundebug v0.0.0-00010101000000-000000000000
	github.com/urfave/cli/v2 v2.3.0
)
