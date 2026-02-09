# Chi HTTP Server

A robust HTTP server implementation using the Chi router with support for CORS and Uber FX dependency injection.

## Features

- 🚀 Built on top of [Chi router](https://github.com/go-chi/chi)
- 🌐 CORS middleware support
- ⚡ Health check endpoint (always enabled at `/healthz`)
- 📊 Prometheus metrics (always enabled on separate port)
- 🔧 Simple and intuitive API
- 📦 Uber FX integration
- 🛡️ Default middleware stack (RequestID, RealIP, Logger, Recoverer)
- ✅ Configuration validation

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

There are only two ways to create a server:

1. **`New(config)`** - Create a server with a configuration
2. **`NewWithLifecycle(config, lc)`** - Create a server with automatic lifecycle management

### Basic Usage

```go
package main

import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
)

func main() {
    // Start with default configuration
    cfg := chi.Default()
    cfg.Port = 8080
    
    // Or build a custom configuration
    cfg = chi.Config{
        Port: 8080,
    }
    
    // Enable CORS if needed
    cfg = cfg.WithDefaultCORS()
    
    server, err := chi.New(cfg)
    if err != nil {
        panic(err)
    }

    // Register routes
    server.Router().Get("/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Start server (blocking)
    if err := server.Start(); err != nil {
        panic(err)
    }
}
```

### With Uber FX

```go
package main

import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        chi.ModuleWithLifecycle,
        fx.Provide(func() chi.Config {
            return chi.Config{
                Port:        3000,
                MetricsPort: 9090,
            }
        }),
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

Prometheus metrics are always enabled on a separate HTTP server:

```go
cfg := chi.Default()
cfg.Port = 8080
cfg.MetricsPort = 9090 // default

server, _ := chi.New(cfg)

// Metrics available at http://localhost:9090/metrics
```

The metrics server runs on a separate port to:
- Avoid exposing metrics on the public API
- Allow different security policies
- Enable metrics collection without affecting main server performance

Metrics are exposed at `/metrics` and include:
- Go runtime metrics (goroutines, memory, GC)
- HTTP request metrics (duration, status codes)
- Custom application metrics (if registered)

### Custom CORS

```go
cfg := chi.Default()
cfg.CORS = &chi.CORSConfig{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           300,
}

server, _ := chi.New(cfg)
```

Or use the convenient method for default CORS:

```go
cfg := chi.Default()
cfg = cfg.WithDefaultCORS()

server, _ := chi.New(cfg)
```

## Configuration

The `Config` struct contains all server settings:

```go
type Config struct {
    Port            uint          // Server port (default: 8080)
    ReadTimeout     time.Duration // Read timeout (default: 15s)
    WriteTimeout    time.Duration // Write timeout (default: 15s)
    IdleTimeout     time.Duration // Idle timeout (default: 60s)
    ShutdownTimeout time.Duration // Graceful shutdown timeout (default: 10s)
    MetricsPort     uint          // Metrics server port (default: 9090)
    CORS            *CORSConfig   // CORS configuration (default: nil)
}
```

**Note:** Health check is always enabled at `/healthz` and metrics are always enabled at `/metrics` on the configured metrics port.

### Getting Default Configuration

```go
cfg := chi.Default()
// Modify as needed
cfg.Port = 3000
```

### Configuration Validation

All configurations are automatically validated before server creation:

```go
cfg := chi.Config{
    Port: 99999, // Invalid!
}

server, err := chi.New(cfg)
// err: invalid server configuration: invalid port: must be between 1 and 65535: 99999
```

## API

### Functions

#### `New(cfg Config) (*Server, error)`
Creates a new HTTP server with the given configuration.

#### `NewWithLifecycle(cfg Config, lc fx.Lifecycle) (*Server, error)`
Creates a new HTTP server with automatic lifecycle management. The server is started when the fx app starts and gracefully shut down when the app stops.

#### `Default() Config`
Returns a configuration with sensible defaults.

### Methods

#### `Router() *chi.Mux`
Returns the Chi router for registering routes.

#### `Start() error`
Starts the HTTP server (blocking call).

#### `Shutdown(ctx context.Context) error`
Gracefully shuts down the server.

#### `Addr() string`
Returns the server address.

#### `MetricsAddr() string`
Returns the metrics server address.

## Health Check

A health check endpoint is always available at `/healthz`:

```bash
curl http://localhost:8080/healthz
# Response: ok
```

## Metrics

Prometheus metrics are always exposed on a separate server at `/metrics`:

```bash
curl http://localhost:9090/metrics
# Response: Prometheus metrics in text format
```

The metrics include:
- Go runtime metrics (goroutines, memory, GC)
- HTTP request metrics (duration, status codes)
- Custom application metrics (if registered)

## Examples

### Complete Example with All Features

```go
package main

import (
    "context"
    "net/http"
    "time"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
)

func main() {
    cfg := chi.Config{
        Port:            8080,
        ReadTimeout:     10 * time.Second,
        WriteTimeout:    10 * time.Second,
        IdleTimeout:     60 * time.Second,
        ShutdownTimeout: 5 * time.Second,
        MetricsPort:     9090,
        CORS: &chi.CORSConfig{
            AllowedOrigins:   []string{"https://example.com"},
            AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
            AllowedHeaders:   []string{"Content-Type", "Authorization"},
            AllowCredentials: true,
            MaxAge:           300,
        },
    }
    
    server, err := chi.New(cfg)
    if err != nil {
        panic(err)
    }
    
    // Register routes
    server.Router().Get("/api/users", getUsers)
    server.Router().Post("/api/users", createUser)
    
    // Start server
    if err := server.Start(); err != nil {
        panic(err)
    }
}

func getUsers(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"users": []}`))
}

func createUser(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"id": 1}`))
}
```

### With FX and Dependency Injection

```go
package main

import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        chi.ModuleWithLifecycle,
        fx.Provide(func() chi.Config {
            return chi.Default().WithDefaultCORS()
        }),
        fx.Invoke(registerRoutes),
    ).Run()
}

func registerRoutes(server *chi.Server) {
    server.Router().Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello from FX!"))
    })
}
```

## Error Handling

The package provides clear error messages for configuration issues:

```go
// Invalid port
cfg := chi.Config{Port: 99999}
_, err := chi.New(cfg)
// err: invalid server configuration: invalid port: must be between 1 and 65535: 99999

// Same port for server and metrics
cfg = chi.Config{
    Port:        8080,
    MetricsPort: 8080,
}
_, err = chi.New(cfg)
// err: invalid server configuration: metrics port must be different from main server port
```

## License

MIT
