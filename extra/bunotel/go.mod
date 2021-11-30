module github.com/uptrace/bun/extra/bunotel

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/uptrace/bun v1.0.19
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.6
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/metric v0.25.0
	go.opentelemetry.io/otel/trace v1.2.0
	golang.org/x/sys v0.0.0-20211124211545-fe61309f8881 // indirect
)
