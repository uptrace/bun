# Example for Bun's OpenTelemetry instrumentation

This example demonstrates how to monitor Bun SQL client using OpenTelemetry and
[Uptrace](https://github.com/uptrace/uptrace). It requires Docker to start PostgreSQL and Uptrace.

See
[SQL performance and errors monitoring](https://bun.uptrace.dev/guide/performance-monitoring.html)
for details.

**Step 1**. Download the example using Git:

```shell
git clone https://github.com/uptrace/bun.git
cd example/opentelemetry
```

**Step 2**. Start the services using Docker:

```shell
docker-compose pull
docker-compose up -d
```

**Step 3**. Make sure Uptrace is running:

```shell
docker-compose logs uptrace
```

**Step 4**. Run the Bun client example:

```shell
UPTRACE_DSN=http://project2_secret_token@localhost:14317/2 go run client.go
```

**Step 5**. Follow the link from the CLI to view the trace:

```shell
UPTRACE_DSN=http://project2_secret_token@localhost:14318?grpc=14317 go run client.go
trace: http://localhost:14318/traces/ee029d8782242c8ed38b16d961093b35
```

![Bun trace](./image/bun-trace.png)

You can also open Uptrace UI at [http://localhost:14318](http://localhost:14318) to view available
spans, logs, and metrics.

## Links

- [Uptrace open-source APM](https://uptrace.dev/get/open-source-apm.html)
- [OpenTelemetry Go instrumentations](https://uptrace.dev/opentelemetry/instrumentations/?lang=go)
- [OpenTelemetry Go Tracing API](https://uptrace.dev/opentelemetry/go-tracing.html)
