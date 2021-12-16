module github.com/uptrace/bun/example/fixture-xid

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

require (
	github.com/rs/xid v1.3.0
	github.com/uptrace/bun v1.0.19
	github.com/uptrace/bun/dbfixture v1.0.19
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.19
	github.com/uptrace/bun/driver/sqliteshim v1.0.19
	github.com/uptrace/bun/extra/bundebug v1.0.19
)
