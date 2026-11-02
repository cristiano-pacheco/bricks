# OpenTelemetry tracing

Add `trace.Module` to an Fx application to load tracing configuration and manage the provider. Declare the tracing module before modules that register application lifecycle hooks. Fx stops hooks in reverse order, so application work can drain before tracing flushes.

```go
fx.New(
    trace.Module,
    application.Module,
).Run()
```

Bricks loads `app.open-telemetry` through `pkg/config` from `config/base.yaml` and the environment-specific YAML file. For example:

```yaml
app:
  open-telemetry:
    app_name: my-service
    app_version: "1.0.0"
    trace_enabled: true
    trace_url: localhost:4317
    exporter_type: grpc
    insecure: true
    sample_rate: 1.0
    batch_timeout: 5s
    max_batch_size: 512
    max_queue_size: 2048
    export_timeout: 10s
    shutdown_timeout: 10s
```

`app_name` is required. An enabled exporter also requires `trace_url`. Use `grpc` or `http` for `exporter_type`; the default is `grpc`. Set `sample_rate` between 0 and 1. A zero rate disables root sampling, while sampled parent contexts remain sampled. When omitted, the rate defaults to 0.01. Sampling does not depend on the application environment.

Batch and queue sizes and timeouts bound export work. Invalid values reject startup. The exporter does not require the backend to be available at startup. A backend outage may cause spans to be dropped. Fx stops tracing within the configured shutdown timeout or the Fx stop deadline, whichever comes first.

Create spans using the existing API:

```go
ctx, span := trace.Span(ctx, "process-order")
defer span.End()
```

Pass the returned context to downstream operations. Use the OpenTelemetry propagator for transport boundaries.

## GORM tracing

Declare `trace.GORMModule()` instead of `trace.Module` and provide a `*gorm.DB` through Fx.
`GORMModule()` includes the tracing module and does not create or close the database client.
Set `app.database.tracing.enabled: true` to install the GORM tracing plugin at
startup. The switch defaults to false and is independent of Redis tracing. When enabled,
a missing client or a client with the plugin already installed rejects startup.
Register application lifecycle hooks after this module so database work begins after
instrumentation is installed and drains before the provider shuts down. Do not modify
GORM callbacks while instrumentation is active. GORM callback registration and removal
are not safe to run concurrently with database operations.

The plugin disables its metrics and query variables and replaces SQL text with an empty
string. Third-party error messages recorded in spans may still contain sensitive data;
these plugin controls do not sanitize those messages.

## Migration

Bricks delegates provider initialization, span creation and shutdown to `github.com/cristiano-pacheco/go-otel`. Applications using `trace.Module` should remove separate initialization and shutdown calls. Existing direct `Initialize`, `MustInitialize`, `Span`, and `Shutdown` APIs remain available for applications that manage their own lifecycle. Do not combine direct initialization with the module.

The module initializes tracing during `app.Start`, rather than during `fx.New`. Spans created by constructors or invokes before startup are not recorded. Move startup instrumentation into lifecycle hooks registered after the tracing module.

The module no longer provides `config.Config[trace.TracerConfig]` to other Fx consumers. Applications that inject that value must provide it separately.

For direct Go configuration, set `SampleRateSet: true` to explicitly select a zero sampling rate. YAML callers only need `sample_rate: 0`. The old environment-based sampling behavior is removed. Set the desired rate explicitly.
