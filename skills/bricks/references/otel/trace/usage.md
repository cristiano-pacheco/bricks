# Export a span with an owned tracing lifecycle

Send one `create-order` span to a local OTLP receiver, then flush and stop. Use an application Go module with Bricks and its supported Go toolchain. You need a running OTLP collector for export, not merely an open TCP port.

## Set up the HTTP exporter

1. Start a collector that enables OTLP/HTTP on port 4318 and OTLP/gRPC on port 4317. The [distributed collector.yaml](../../../examples/tracing-span/collector.yaml) sends received spans to its debug exporter. For a local Docker setup, copy that file into your working directory and run:

```sh
docker run --rm --name bricks-trace-collector \
  -p 127.0.0.1:4317:4317 -p 127.0.0.1:4318:4318 \
  -v "$PWD/collector.yaml:/etc/otelcol/config.yaml:ro" \
  otel/opentelemetry-collector:0.123.0 --config=/etc/otelcol/config.yaml
```

   The collector is an external service, not a Bricks dependency created by the module. Leave its terminal open to see spans. This Docker command may download the collector image when you run it.

2. Create `cmd/order-trace/config/base.yaml` in your application:

```yaml
app:
  open-telemetry:
    app_name: orders
    trace_enabled: true
    trace_url: localhost:4318
    exporter_type: http
    insecure: true
    sample_rate: 1.0
```

   `trace_url` is `host:port`, without a scheme or `/v1/traces` path. The HTTP exporter supplies the default OTLP trace path. `insecure: true` selects plaintext for this local example. Use TLS for an appropriate production receiver. The explicit sample rate makes this demonstration deterministic for a root span.

3. Create `cmd/order-trace/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"go.uber.org/fx"
)

func registerOrder(lifecycle fx.Lifecycle) {
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return createOrder(ctx)
	}})
}

func createOrder(ctx context.Context) error {
	ctx, span := brickstrace.Span(ctx, "create-order")
	defer span.End()
	return ctx.Err()
}

func run() error {
	app := fx.New(brickstrace.Module, fx.Invoke(registerOrder))
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	err := app.Start(startCtx)
	cancelStart()
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

   Declare the tracing module before application lifecycle hooks. Fx initializes tracing during `app.Start`, not during constructors or invokes. Application work starts after initialization. Reverse stop order lets application shutdown finish before tracing flushes. End every span and pass its returned context into downstream work.

4. Run from the application root:

```sh
APP_CONFIG_DIR=cmd/order-trace/config APP_ENV=local go run ./cmd/order-trace
```

   Expect the program to stop after one operation. The collector debug output should contain a span named `create-order` and the resource attribute `service.name: orders`. There is no HTTP response to scrape. Successful application startup alone does not prove delivery. Export is asynchronous, a receiver outage can drop spans, and export errors may appear through OpenTelemetry's error handler rather than the business operation's return value.

   Copy the [distributed main.go](../../../examples/tracing-span/main.go) and [base.yaml](../../../examples/tracing-span/config/base.yaml) instead of transcribing them if preferred. 

## Select a different scenario

For OTLP/gRPC, replace `base.yaml` with this complete alternative, also [distributed](../../../examples/tracing-span/config/grpc.yaml). The program and shutdown ownership stay the same.

```yaml
app:
  open-telemetry:
    app_name: orders
    trace_enabled: true
    trace_url: localhost:4317
    exporter_type: grpc
    insecure: true
    sample_rate: 1.0
```

For disabled export, use [disabled.yaml](../../../examples/tracing-span/config/disabled.yaml):

```yaml
app:
  open-telemetry:
    app_name: orders
    trace_enabled: false
```

The module still requires a nonempty application name and initializes a non-recording provider. It does not create an exporter. `Span` remains safe to end. Before initialization, `Span` returns the original context and a no-op span without ending the existing parent.

## Complete configuration

The module loads `app.open-telemetry` from `base.yaml` plus optional `<APP_ENV>.yaml` in `APP_CONFIG_DIR`. The defaults are `APP_ENV=local` and `APP_CONFIG_DIR=config`. An exact `env://NAME` string resolves an environment variable through the standard configuration loader. The distributed [full.yaml](../../../examples/tracing-span/config/full.yaml) contains every supported module field:

```yaml
app:
  open-telemetry:
    app_name: orders
    app_version: "1.0.0"
    tracer_vendor: otlp
    trace_enabled: true
    trace_url: localhost:4318
    exporter_type: http
    insecure: true
    sample_rate: 1.0
    batch_timeout: 5s
    max_batch_size: 512
    max_queue_size: 2048
    export_timeout: 10s
    shutdown_timeout: 10s
```

To select a reference file as an overlay, set `APP_ENV=full`, `grpc`, or `disabled`. The base file remains required. For an independent configuration, copy the selected file to `base.yaml` in a new directory.

| Field | Type | Required, default, and constraints |
| --- | --- | --- |
| `app_name` | string | Required even when disabled. Nonempty. Sets `service.name`. |
| `app_version` | string | Optional, empty. Sets `service.version`. |
| `tracer_vendor` | string | Optional, empty. Accepted and forwarded, but currently has no effect in the delegated provider. It does not select an exporter. |
| `trace_enabled` | bool | Optional, false. False selects a never-sample provider without export. |
| `trace_url` | string | Required only when enabled. OTLP endpoint host and port, no scheme or path. |
| `exporter_type` | string | Optional, `grpc`. Only `grpc` and `http` are accepted. |
| `insecure` | bool | Optional, false. True disables transport security; not certificate verification alone. |
| `sample_rate` | float | Optional, 0.01. Inclusive range 0 to 1. Explicit YAML zero disables root sampling. Parent-based sampling still honors sampled parents when enabled. NaN is invalid. |
| `batch_timeout` | duration | Optional, zero becomes 5s. Maximum batch wait; negative values fail validation. |
| `max_batch_size` | int | Optional, zero becomes 512. Maximum export batch. Must not exceed the effective queue size. Negative values fail. |
| `max_queue_size` | int | Optional, zero becomes 2048. Maximum queued spans. Negative values fail. Queue pressure can drop spans. |
| `export_timeout` | duration | Optional, zero becomes 10s. Bounds exporter work. Negative values fail. |
| `shutdown_timeout` | duration | Optional, zero becomes 10s. Bounds provider cleanup. Negative values fail. |

Use duration strings with units such as `500ms` or `10s`. Disabled mode still validates configuration limits and sampling values. Exporter parsing also rejects unsupported names before startup. The module uses a private YAML type and does not provide `config.Config[trace.TracerConfig]` to other consumers. Do not invent a `sample_rate_set` YAML field. YAML presence already distinguishes an explicit zero.

## Direct construction

Without Fx, use `Initialize`, call your application function, then call `Shutdown` with a fresh bounded context. This fragment belongs in a function that returns an error and already imports `context`, `errors`, `time`, and `brickstrace` as in the full example:

```go
exporter, err := brickstrace.NewExporterType(brickstrace.ExporterTypeHTTP)
if err != nil { return err }
if err := brickstrace.Initialize(brickstrace.TracerConfig{
    AppName: "orders", TraceEnabled: true, TraceURL: "localhost:4318",
    ExporterType: exporter, Insecure: true,
    SampleRate: 1, SampleRateSet: true,
}); err != nil { return err }
workErr := createOrder(context.Background())
stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
return errors.Join(workErr, brickstrace.Shutdown(stopCtx))
```

`TracerConfig.Validate()` checks required values and limits. `NewExporterType` returns the value type also exposed as `ExporterType`; it supports `String`, `IsGRPC`, `IsHTTP`, and `IsZero`. Its zero value selects gRPC at initialization. In Go, use `SampleRateSet: true` to make a zero `SampleRate` explicit. Nonzero rates work without that flag. `InitializeWithLifecycle` accepts `config.Config[TracerConfig]` and appends the same Fx lifecycle hooks when you own configuration provisioning. It does not perform the module's string-to-`ExporterType` conversion or its pointer-based sampling conversion. Prefer `Module` for the documented YAML contract. The public `TracerConfig` is the direct Go configuration type; its `ExporterType` field is a value object, not a string.

Tracing is process-global. Do not combine direct initialization with `Module` or initialize concurrent independent providers through this package. Reinitialization before shutdown returns `ErrAlreadyInitialized`. `IsInitialized()` reports provider ownership state, not backend connectivity. `Shutdown` is safe before initialization and on repeated calls. It resets the global provider to no-op and bounds cleanup by its configured budget. A caller deadline can return before bounded background cleanup completes.

`MustInitialize` panics on failure and is not the recommended application entry point. Propagate errors from `Initialize`. Exported errors cover invalid required fields, limits, sampling and exporter type, initialization state, exporter/provider creation, and shutdown. Use `errors.Is` with the relevant exported sentinel and preserve wrapped context. Some sentinels remain public for compatibility and are not emitted by every current path.

## Add transport or storage instrumentation only when needed

For transport propagation, extract inbound and inject outbound headers with `otel.GetTextMapPropagator()`. Initialization installs W3C TraceContext and Baggage propagators. Creating a span does not automatically propagate across HTTP or Redis boundaries.

`GORMModule()` includes `Module`. Use it instead of `Module`, supply an initialized `*gorm.DB`, and enable `app.database.tracing.enabled`. It does not create or close the database. It installs callbacks at startup and removes its callbacks at shutdown. Existing tracing plugins reject startup. Do not change callbacks concurrently with queries. SQL text and query variables are suppressed, but third-party error text can still contain sensitive data.

`RedisModule()` does not include the base provider module. Add it alongside `Module` or `GORMModule()` and supply `redis.UniversalClient`. Enable `app.redis.tracing.enabled`. It accepts only a nonnil underlying `*redis.Client`, including compatible failover clients. Cluster and ring clients are rejected. Shutdown disables its retained hook because go-redis has no hook-removal API; client closure stays with the client owner. Both add-on switches default false and are Go-composed instrumentation choices, not exporter selectors. Detailed storage wiring belongs in the storage tracing recipes.
