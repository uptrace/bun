module github.com/uptrace/bun/extra/bunotel

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/uptrace/bun v1.0.8
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/metric v0.23.0
	go.opentelemetry.io/otel/trace v1.0.0-RC3
)
