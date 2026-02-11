# Chi HTTP Server

HTTP server baseado em Chi com CORS, Prometheus e FX.

## Features

- Chi router, CORS, middleware padrão (RequestID, RealIP, Logger, Recoverer)
- Health `/healthz`, métricas Prometheus em porta separada
- `Route`: interface para módulos registrarem rotas via FX group
- Validação de config

## Configuração

```go
type Config struct {
    Port            uint          // default: 8080
    ReadTimeout     time.Duration // default: 15s
    WriteTimeout    time.Duration // default: 15s
    IdleTimeout     time.Duration // default: 60s
    ShutdownTimeout time.Duration // default: 10s
    MetricsPort     uint          // default: 9090
    CORS            *CORSConfig
}
```

## Uso

### Sem FX

```go
cfg := chi.Default()
server, _ := chi.New(cfg)

server.Router().Get("/hello", handler)

// Ou: server.RegisterRoute(rota); server.SetupRoutes()
server.Start()
```

### Com FX

**1. Módulo servidor (app):**

```go
fx.Module(
    "httpserver",
    config.Provide[chi.Config]("app.http"),
    chi.Module,
    fx.Invoke(func(*chi.Server) {}), // força construção
)
```

**2. Router implementa `chi.Route`:**

```go
type ContactRouter struct { 
    handler *handler.ContactHandler 
}

func (r *ContactRouter) Setup(server *chi.Server) {
    r := server.Router()
    r.Get("/api/contacts", r.handler.List)
}
```

**3. Registrar no FX:**

```go
fx.Annotate(
    router.NewContactRouter,
    fx.As(new(chi.Route)),
    fx.ResultTags(`group:"routes"`),
)
```

**4. App principal:**

```go
fx.New(httpserver.Module, monitor.Module).Run()
```

Rotas do group `"routes"` são coletadas automaticamente; servidor inicia no OnStart e dá shutdown no OnStop.

## API

| Função/Método | Descrição |
|---------------|-----------|
| `New(cfg)` | Cria servidor |
| `NewWithLifecycle(params)` | Cria com lifecycle FX (config, lc, routes, logger opcional) |
| `Default()` | Config com defaults |
| `Router()` | Chi Mux |
| `RegisterRoute(r)`, `RegisterRoutes(routes)` | Adiciona ao registry |
| `SetupRoutes()` | Chama Setup em todas as rotas (antes de Start) |
| `Start()`, `Shutdown(ctx)` | Ciclo de vida |
| `Addr()`, `MetricsAddr()` | Endereços |

## CORS

```go
cfg.CORS = &chi.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST"},
}
// ou: cfg = cfg.WithDefaultCORS()
```

## Endpoints

- `/healthz` — health check
- `/metrics` — Prometheus (porta MetricsPort)
