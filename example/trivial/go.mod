module github.com/uptrace/bun/example/trivial

go 1.23.0

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/dialect/mysqldialect => ../../dialect/mysqldialect

replace github.com/uptrace/bun/dialect/mssqldialect => ../../dialect/mssqldialect

require (
	github.com/denisenkom/go-mssqldb v0.12.3
	github.com/go-sql-driver/mysql v1.9.3
	github.com/uptrace/bun v1.2.14
	github.com/uptrace/bun/dialect/mssqldialect v1.2.14
	github.com/uptrace/bun/dialect/mysqldialect v1.2.14
	github.com/uptrace/bun/dialect/pgdialect v1.2.14
	github.com/uptrace/bun/dialect/sqlitedialect v1.2.14
	github.com/uptrace/bun/driver/pgdriver v1.2.14
	github.com/uptrace/bun/driver/sqliteshim v1.2.14
	github.com/uptrace/bun/extra/bundebug v1.2.14
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250606033433-dcc06ee1d476 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	mellium.im/sasl v0.3.2 // indirect
	modernc.org/libc v1.65.10 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.38.0 // indirect
)
