module github.com/uptrace/bun/example/db2

go 1.24.0

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dialect/db2dialect => ../../dialect/db2dialect

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

require (
	github.com/ibmdb/go_ibm_db v0.5.4
	github.com/uptrace/bun v1.3.0
	github.com/uptrace/bun/dialect/db2dialect v1.3.0
	github.com/uptrace/bun/extra/bundebug v1.3.0
)

require (
	github.com/fatih/color v1.18.0 // indirect
	github.com/ibmruntimes/go-recordio/v2 v2.0.0-20240416213906-ae0ad556db70 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)
