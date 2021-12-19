module github.com/uptrace/bun/extra/bunotel

go 1.16

replace github.com/uptrace/bun => ../..

require (
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/uptrace/bun v1.0.20
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.7
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/metric v0.26.0
	go.opentelemetry.io/otel/trace v1.3.0
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
)
