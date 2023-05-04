module github.com/TommyLeng/bun/example/opentelemetry

go 1.18

replace github.com/TommyLeng/bun => ../..

replace github.com/TommyLeng/bun/dbfixture => ../../dbfixture

replace github.com/TommyLeng/bun/extra/bundebug => ../../extra/bundebug

replace github.com/TommyLeng/bun/extra/bunotel => ../../extra/bunotel

replace github.com/TommyLeng/bun/dialect/pgdialect => ../../dialect/pgdialect

replace github.com/TommyLeng/bun/driver/pgdriver => ../../driver/pgdriver

require (
	github.com/brianvoe/gofakeit/v5 v5.11.2
	github.com/TommyLeng/bun v1.1.12
	github.com/TommyLeng/bun/dialect/pgdialect v1.1.12
	github.com/TommyLeng/bun/driver/pgdriver v1.1.12
	github.com/TommyLeng/bun/extra/bunotel v1.1.12
	github.com/uptrace/uptrace-go v1.13.0
	go.opentelemetry.io/otel v1.13.0
)

require (
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.21 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.13.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.13.0 // indirect
	go.opentelemetry.io/otel/metric v0.36.0 // indirect
	go.opentelemetry.io/otel/sdk v1.13.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.13.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20230216225411-c8e22ba71e44 // indirect
	google.golang.org/grpc v1.53.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	mellium.im/sasl v0.3.1 // indirect
)
