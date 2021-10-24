module github.com/uptrace/bun/extra/bunotel

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/uptrace/bun v1.0.14
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/trace v1.0.1
)
