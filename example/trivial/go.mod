module github.com/uptrace/bun/example/trivial

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/uptrace/bun v1.0.3
	github.com/uptrace/bun/dialect/mysqldialect v1.0.3
	github.com/uptrace/bun/dialect/pgdialect v1.0.3
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.3
	github.com/uptrace/bun/driver/pgdriver v1.0.3
	github.com/uptrace/bun/driver/sqliteshim v1.0.3
	github.com/uptrace/bun/extra/bundebug v1.0.3
)
