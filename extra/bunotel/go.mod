module github.com/uptrace/bun/extra/bunotel

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/uptrace/bun v1.0.0-rc.3
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC1
)
