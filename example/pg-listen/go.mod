module github.com/TommyLeng/bun/example/pg-listen

go 1.18

replace github.com/TommyLeng/bun => ../..

replace github.com/TommyLeng/bun/extra/bundebug => ../../extra/bundebug

replace github.com/TommyLeng/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/TommyLeng/bun/dialect/pgdialect => ../../dialect/pgdialect

require (
	github.com/TommyLeng/bun v1.1.12
	github.com/TommyLeng/bun/dialect/pgdialect v1.1.12
	github.com/TommyLeng/bun/driver/pgdriver v1.1.12
	github.com/TommyLeng/bun/extra/bundebug v1.1.12
)

require (
	github.com/fatih/color v1.14.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	mellium.im/sasl v0.3.1 // indirect
)
