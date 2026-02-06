# Chi HTTP Server

A robust HTTP server implementation using the Chi router with support for CORS and Uber FX dependency injection.

## Features

- 🚀 Built on top of [Chi router](https://github.com/go-chi/chi)
- 🌐 CORS middleware support
- ⚡ Health check endpoint
- � Prometheus metrics on separate port
- �🔧 Functional options pattern
- 📦 Uber FX integration
- 🛡️ Default middleware stack (RequestID, RealIP, Logger, Recoverer)

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

### Basic Usage

```go
package main

import (
    "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
)

func main() {
    server, err := chi.New(
        chi.WithHost("0.0.0.0"),
        chi.WithPort(8080),
        chi.WithDefaultCORS(),
    )
    if err != nil {
        panic(err)
    }

    // Register routes
    server.Router().Get("/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Start server
    if err := server.Start(); err != nil {
        panic(err)
    }
}
```

### With Uber FX

```go
package main

import (
    "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        chi.ModuleWithLifecycle,
        chi.ProvideOptions(
            chi.WithHost("0.0.0.0"),
            chi.WithPort(3000),
            chi.WithDefaultCORS(),
        ),
        fx.Invoke(registerRoutes),
    ).Run()
}

func registerRoutes(server *chi.Server) {
    server.Router().Get("/api/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello from FX!"))
    })
}
```

### Prometheus Metrics

Enable Prometheus metrics on a separate HTTP server:

```go
server, _ := chi.New(
    chi.WithHost("0.0.0.0"),
    chi.WithPort(8080),
    chi.WithMetrics(true),           // Enable metrics endpoint
    chi.WithMetricsPort(9090),        // Metrics on port 9090 (default)
    chi.WithMetricsPath("/metrics"),  // Metrics path (default)
)

// Metrics available at http://localhost:9090/metrics
```

The metrics server runs on a separate port to:
- Avoid exposing metrics on the public API
- Allow different security policies
- Enable metrics collection without affecting main server performance

### Custom CORS

```go
server, _ := chi.New(
    chi.WithCORS(&chi.CORSConfig{
        AllowedOrigins:   []string{"https://example.com"},
        AllowedMethods:   []string{"GET", "POST"},
        AllowedHeaders:   []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           300,
    }),
)
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithHost` | Server host | `localhost` |
| `WithPort` | Server port | `8080` |
| `WithReadTimeout` | Read timeout | `15s` |
| `WithWriteTimeout` | Write timeout | `15s` |
| `WithIdleTimeout` | Idle timeout | `60s` |
| `WithShutdownTimeout` | Graceful shutdown timeout | `10s` |
| `WithHealthCheck` | Enable health check | `true` |
| `WithHealthCheckPath` | Health check endpoint | `/healthz` |
| `WithMetrics` | Enable Prometheus metrics | `false` |
| `WithMetricsPort` | Metrics server port | `9090` |
| `WithMetricsPath` | Metrics endpoint path | `/metrics` |
| `WithCORS` | Custom CORS config | `nil` |
| `WithDefaultCORS` | Permissive CORS config | - |

## Health Check

By default, a health check endpoint is available at `/healthz`:

```bash
curl http://localhost:8080/healthz
# RMetrics

When enabled, Prometheus metrics are exposed on a separate server at `/metrics`:

```bash
curl http://localhost:9090/metrics
# Response: Prometheus metrics in text format
```

The metrics include:
- Go runtime metrics (goroutines, memory, GC)
- HTTP request metrics (duration, status codes)
- Custom application metrics (if registered)

## esponse: ok
```

## License

MIT
