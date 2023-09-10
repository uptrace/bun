module github.com/uptrace/bun/example/pg-listen

go 1.19

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

require (
	github.com/uptrace/bun v1.1.15
	github.com/uptrace/bun/dialect/pgdialect v1.1.15
	github.com/uptrace/bun/driver/pgdriver v1.1.15
	github.com/uptrace/bun/extra/bundebug v1.1.15
)

require (
	github.com/fatih/color v1.15.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	mellium.im/sasl v0.3.1 // indirect
)
