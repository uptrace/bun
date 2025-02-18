module github.com/uptrace/bun/internal/dbtest

go 1.23.0

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/dialect/mssqldialect => ../../dialect/mssqldialect

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

require (
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible
	github.com/brianvoe/gofakeit/v6 v6.4.1
	github.com/denisenkom/go-mssqldb v0.12.2
	github.com/go-sql-driver/mysql v1.6.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v4 v4.11.0
	github.com/stretchr/testify v1.10.0
	github.com/uptrace/bun v1.2.10
	github.com/uptrace/bun/dbfixture v1.2.10
	github.com/uptrace/bun/dialect/mssqldialect v1.2.10
	github.com/uptrace/bun/dialect/mysqldialect v1.2.10
	github.com/uptrace/bun/dialect/pgdialect v1.2.10
	github.com/uptrace/bun/dialect/sqlitedialect v1.2.10
	github.com/uptrace/bun/driver/pgdriver v1.2.10
	github.com/uptrace/bun/driver/sqliteshim v1.2.10
	github.com/uptrace/bun/extra/bundebug v1.2.10
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.8.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.6 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.7.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/exp v0.0.0-20250215185904-eff6e970281f // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mellium.im/sasl v0.3.2 // indirect
	modernc.org/libc v1.61.13 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.2 // indirect
	modernc.org/sqlite v1.35.0 // indirect
)
