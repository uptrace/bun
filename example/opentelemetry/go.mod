module github.com/uptrace/bun/example/basic

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/extra/bunotel => ../../extra/bunotel

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

require (
	github.com/brianvoe/gofakeit/v5 v5.11.2
	github.com/mattn/go-sqlite3 v1.14.9 // indirect
	github.com/uptrace/bun v1.0.16
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.16
	github.com/uptrace/bun/driver/sqliteshim v1.0.16
	github.com/uptrace/bun/extra/bunotel v1.0.16
	github.com/uptrace/opentelemetry-go-extra/otelplay v0.1.4
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.4 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	go.opentelemetry.io/otel v1.1.0
	golang.org/x/net v0.0.0-20211105192438-b53810dc28af // indirect
	golang.org/x/sys v0.0.0-20211107104306-e0b2ad06fe42 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	modernc.org/ccgo/v3 v3.12.54 // indirect
	modernc.org/sqlite v1.13.3 // indirect
)
