module github.com/uptrace/bun/extra/bunrelic

go 1.23.0 // required by google.golang.org/grpc v1.68.0

replace github.com/uptrace/bun => ../..

require (
	github.com/newrelic/go-agent/v3 v3.40.0
	github.com/uptrace/bun v1.2.15
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/grpc v1.74.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
