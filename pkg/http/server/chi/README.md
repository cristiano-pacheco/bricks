# Chi HTTP Server

HTTP server based on Chi with CORS, Prometheus and FX.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

To use the module, add it to your fx app:

```go
app := fx.New(chi.Module)
app.Run()
```

## Features

- Chi router, CORS, and configurable middleware (RequestID, RealIP, Logger, Recoverer)
- Health `/healthz`, with an optional Prometheus listener
- Swagger `/swagger/` endpoint for Swagger
- `Route`: interface for modules to register routes via FX
- Synchronous listener binding with support for port `0`
- Config validation

## Configuration

```go
type Config struct {
    Address               string        // complete listen address; empty uses 0.0.0.0:Port
    Port                  uint          // default: 8080; port 0 selects an available port
    ReadTimeout           time.Duration // default: 15s
    WriteTimeout          time.Duration // default: 15s
    IdleTimeout           time.Duration // default: 60s
    ShutdownTimeout       time.Duration // default: 10s
    EnableMetrics         bool          // default: true
    MetricsAddress        string        // complete metrics listen address
    MetricsPort           uint          // default: 9090; port 0 selects an available port
    EnableRequestLogging  bool          // default: true
    EnableRouteLogging    bool          // default: true
    CORS                  *CORSConfig
}
```

## Usage

### Without FX

```go
cfg := chi.Default()
server, _ := chi.New(cfg)

server.Router().Get("/hello", handler)

// Or: server.RegisterRoute(route); server.SetupRoutes()
if err := server.Start(); err != nil {
    log.Fatal(err)
}
```

### With FX

**1. Server module (app):**

```go
fx.Module(
    "httpserver",
    config.Provide[chi.Config]("app.http"),
    chi.Module,
    fx.Invoke(func(*chi.Server) {}), // forces construction
)
```

**2. Router implements `chi.Route`:**

```go
type ContactRouter struct { 
    handler *handler.ContactHandler 
}

func (r *ContactRouter) Setup(server *chi.Server) {
    r := server.Router()
    r.Get("/api/contacts", r.handler.List)
}
```

**3. Register in FX:**

```go
fx.Annotate(
    router.NewContactRouter,
    fx.As(new(chi.Route)),
    fx.ResultTags(`group:"routes"`),
)
```

**4. Main app:**

```go
fx.New(httpserver.Module, monitor.Module).Run()
```

Routes from the `"routes"` group are collected automatically; the server starts on OnStart and shuts down on OnStop.

## API

| Function/Method | Description |
|-----------------|-------------|
| `New(cfg)` | Creates a server |
| `NewWithLifecycle(params)` | Creates with FX lifecycle (config, lc, routes, optional logger) |
| `Default()` | Config with defaults |
| `Router()` | Chi Mux |
| `RegisterRoute(r)`, `RegisterRoutes(routes)` | Adds to the registry |
| `SetupRoutes()` | Calls Setup on all routes (before Start) |
| `Start()`, `Shutdown(ctx)` | Binds listeners and serves until shutdown |
| `Addr()`, `MetricsAddr()` | Configured or effective listener addresses |

## CORS

```go
cfg.CORS = &chi.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST"},
}
// or: cfg = cfg.WithDefaultCORS()
```

## Endpoints

- `/healthz` — health check
- `/metrics` — Prometheus when `EnableMetrics` is true
