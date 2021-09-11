module github.com/uptrace/bun/internal/dbtest

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

require (
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible
	github.com/brianvoe/gofakeit/v6 v6.4.1
	github.com/go-sql-driver/mysql v1.5.0
	github.com/google/uuid v1.0.0
	github.com/jackc/pgx/v4 v4.11.0
	github.com/stretchr/testify v1.7.0
	github.com/uptrace/bun v1.0.6
	github.com/uptrace/bun/dbfixture v1.0.6
	github.com/uptrace/bun/dialect/mysqldialect v1.0.6
	github.com/uptrace/bun/dialect/pgdialect v1.0.6
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.6
	github.com/uptrace/bun/driver/pgdriver v1.0.6
	github.com/uptrace/bun/driver/sqliteshim v1.0.6
	github.com/uptrace/bun/extra/bundebug v1.0.6
)
