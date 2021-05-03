module github.com/uptrace/bun/internal/dbtest

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

require (
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible
	github.com/go-mysql-org/go-mysql v1.1.2
	github.com/jackc/pgx/v4 v4.11.0
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v0.0.0-00010101000000-000000000000
	github.com/uptrace/bun/dialect/mysqldialect v0.0.0-00010101000000-000000000000
	github.com/uptrace/bun/dialect/pgdialect v0.0.0-00010101000000-000000000000
	github.com/uptrace/bun/dialect/sqlitedialect v0.0.0-00010101000000-000000000000
)
