module github.com/uptrace/bun/example/opentelemetry

go 1.16

replace github.com/uptrace/bun => ../..

replace github.com/uptrace/bun/dbfixture => ../../dbfixture

replace github.com/uptrace/bun/extra/bundebug => ../../extra/bundebug

replace github.com/uptrace/bun/extra/bunotel => ../../extra/bunotel

replace github.com/uptrace/bun/dialect/sqlitedialect => ../../dialect/sqlitedialect

replace github.com/uptrace/bun/driver/sqliteshim => ../../driver/sqliteshim

require (
	github.com/brianvoe/gofakeit/v5 v5.11.2
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/uptrace/bun v1.0.20
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.20
	github.com/uptrace/bun/driver/sqliteshim v1.0.20
	github.com/uptrace/bun/extra/bunotel v1.0.20
	github.com/uptrace/opentelemetry-go-extra/otelplay v0.1.7
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.7 // indirect
	go.opentelemetry.io/otel v1.3.0
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	google.golang.org/grpc v1.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
