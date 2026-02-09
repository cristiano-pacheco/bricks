# OpenTelemetry Trace

Simple and powerful OpenTelemetry tracing integration for Go applications.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Quick Start

### With Uber FX

```go
package main

import (
    "time"
    
    "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        trace.Module,
        fx.Provide(NewTracerConfig),
    ).Run()
}

func NewTracerConfig() trace.TracerConfig {
    exporterType, _ := trace.NewExporterType(trace.ExporterTypeGRPC)
    
    return trace.TracerConfig{
        AppName:      "my-service",
        TraceEnabled: true,
        TracerVendor: "jaeger",
        TraceURL:     "localhost:4317",
        ExporterType: exporterType,
        Insecure:     true,
        SampleRate:   1.0,
        BatchTimeout: 5 * time.Second,
        MaxBatchSize: 512,
    }
}
```

### Without FX

```go
package main

import (
    "context"
    "time"
    
    "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
)

func main() {
    exporterType, _ := trace.NewExporterType(trace.ExporterTypeGRPC)
    
    config := trace.TracerConfig{
        AppName:      "my-service",
        TraceEnabled: true,
        TracerVendor: "jaeger",
        TraceURL:     "localhost:4317",
        ExporterType: exporterType,
        Insecure:     true,
        SampleRate:   1.0,
        BatchTimeout: 5 * time.Second,
        MaxBatchSize: 512,
    }

    trace.MustInitialize(config)
    defer trace.Shutdown(context.Background())

    // Your application logic...
}
```

## Usage

### Creating Spans

```go
func ProcessOrder(ctx context.Context, orderID string) error {
    ctx, span := trace.Span(ctx, "ProcessOrder")
    defer span.End()

    span.SetAttributes(
        attribute.String("order.id", orderID),
    )

    // Your business logic...
    
    return nil
}
```

### HTTP Middleware

```go
func TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, span := trace.Span(r.Context(), r.Method+" "+r.URL.Path)
        defer span.End()

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Configuration

```go
type TracerConfig struct {
    AppName      string        // Application name (required)
    TraceEnabled bool          // Enable/disable tracing
    TracerVendor string        // Tracer vendor (jaeger, tempo, etc.)
    TraceURL     string        // OTLP endpoint (required if enabled)
    ExporterType ExporterType  // ExporterTypeGRPC or ExporterTypeHTTP
    BatchTimeout time.Duration // Batch timeout (default: 5s)
    MaxBatchSize int           // Max spans per batch (default: 512)
    Insecure     bool          // Use insecure connection
    SampleRate   float64       // Sampling rate 0.0-1.0 (default: 1.0)
}
```

### YAML Example

```yaml
app:
  name: myapp

opentelemetry:
  enabled: true
  tracervendor: jaeger
  tracerurl: localhost:4317
  exportertype: grpc
  batchtimeout: 5s
  maxbatchsize: 512
  insecure: true
  samplerate: 1.0
```

## Environment Examples

### Development (Local Jaeger)

```go
exporterType, _ := trace.NewExporterType(trace.ExporterTypeGRPC)

trace.TracerConfig{
    AppName:      "my-service",
    TraceEnabled: true,
    TracerVendor: "jaeger",
    TraceURL:     "localhost:4317",
    ExporterType: exporterType,
    Insecure:     true,
    SampleRate:   1.0,
}
```

### Production

```go
exporterType, _ := trace.NewExporterType(trace.ExporterTypeGRPC)

trace.TracerConfig{
    AppName:      "my-service",
    TraceEnabled: true,
    TracerVendor: "tempo",
    TraceURL:     "tempo.production.com:443",
    ExporterType: exporterType,
    Insecure:     false,
    SampleRate:   0.1, // 10% sampling
}
```

### HTTP Exporter

```go
exporterType, _ := trace.NewExporterType(trace.ExporterTypeHTTP)

trace.TracerConfig{
    AppName:      "my-service",
    TraceEnabled: true,
    TraceURL:     "localhost:4318",
    ExporterType: exporterType,
    Insecure:     true,
}
```

## Running Jaeger Locally

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

Access UI at http://localhost:16686

## Best Practices

- Always defer `span.End()`
- Use descriptive span names
- Add relevant attributes
- Record errors with `span.RecordError(err)`
- Use appropriate sampling rates in production
- Always propagate context

## License

Part of the bricks project.
