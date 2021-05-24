module github.com/uptrace/bun/internal/dbtest

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

require (
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible
	github.com/brianvoe/gofakeit/v6 v6.4.1
	github.com/go-sql-driver/mysql v1.5.0
	github.com/google/uuid v1.0.0
	github.com/jackc/pgx/v4 v4.11.0
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v0.1.1
	github.com/uptrace/bun/dbfixture v0.0.0-00010101000000-000000000000
	github.com/uptrace/bun/dialect/mysqldialect v0.1.0
	github.com/uptrace/bun/dialect/pgdialect v0.1.0
	github.com/uptrace/bun/dialect/sqlitedialect v0.1.0
	github.com/uptrace/bun/driver/pgdriver v0.1.0
	github.com/uptrace/bun/extra/bundebug v0.0.0-00010101000000-000000000000
)
