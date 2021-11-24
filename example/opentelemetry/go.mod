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
	github.com/uptrace/bun v1.0.18
	github.com/uptrace/bun/dialect/sqlitedialect v1.0.18
	github.com/uptrace/bun/driver/sqliteshim v1.0.18
	github.com/uptrace/bun/extra/bunotel v1.0.18
	github.com/uptrace/opentelemetry-go-extra/otelplay v0.1.5
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.5 // indirect
	go.opentelemetry.io/otel v1.2.0
	golang.org/x/net v0.0.0-20211123203042-d83791d6bcd9 // indirect
	google.golang.org/genproto v0.0.0-20211118181313-81c1377c94b1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
