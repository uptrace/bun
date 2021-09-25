module github.com/uptrace/bun/example/fixture

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/mattn/go-colorable v0.1.10 // indirect
	github.com/uptrace/bun v1.0.8
	github.com/uptrace/bun/dbfixture v1.0.8
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.8
	github.com/uptrace/bun/driver/sqliteshim v1.0.8
	github.com/uptrace/bun/extra/bundebug v1.0.8
)
