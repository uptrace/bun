module github.com/TommyLeng/bun/example/migrate

go 1.18

replace github.com/TommyLeng/bun => ../..

replace github.com/TommyLeng/bun/extra/bundebug => ../../extra/bundebug

replace github.com/TommyLeng/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/TommyLeng/bun/driver/sqliteshim => ../../driver/sqliteshim

require (
	github.com/TommyLeng/bun v1.1.12
	github.com/TommyLeng/bun/dialect/sqlitedialect v1.1.12
	github.com/TommyLeng/bun/driver/sqliteshim v1.1.12
	github.com/TommyLeng/bun/extra/bundebug v1.1.12
	github.com/urfave/cli/v2 v2.24.4
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.14.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	lukechampine.com/uint128 v1.2.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.13 // indirect
	modernc.org/libc v1.22.2 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/sqlite v1.20.4 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.1.0 // indirect
)
