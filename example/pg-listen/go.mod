module github.com/uptrace/bun/example/pg-listen

go 1.22.0

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/driver/pgdriver => ../../driver/pgdriver

replace github.com/uptrace/bun/dialect/pgdialect => ../../dialect/pgdialect

require (
	github.com/uptrace/bun v1.2.10
	github.com/uptrace/bun/dialect/pgdialect v1.2.10
	github.com/uptrace/bun/driver/pgdriver v1.2.10
	github.com/uptrace/bun/extra/bundebug v1.2.10
)

require (
	github.com/fatih/color v1.18.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	mellium.im/sasl v0.3.2 // indirect
)
