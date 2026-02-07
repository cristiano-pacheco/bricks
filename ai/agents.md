# Bricks - LLM Context Guide

## Project Overview

**Bricks** is a Go framework providing reusable, production-ready building blocks for web applications. The project emphasizes:
- **Uber FX Integration**: First-class dependency injection support across all modules
- **Type Safety**: Extensive use of Go generics for type-safe APIs
- **Modern Tooling**: Built on modern libraries (Koanf, Zap, GORM, Chi)
- **Production Ready**: Battle-tested patterns with comprehensive error handling

**Module**: `github.com/cristiano-pacheco/bricks`  
**Go Version**: 1.25.7  
**License**: MIT

## Architecture & Design Philosophy

### Core Principles

1. **FX-First Design**: All modules provide FX constructors for seamless dependency injection
2. **Options Pattern**: Flexible configuration using functional options
3. **Generics & Type Safety**: Leverage Go generics for compile-time safety (e.g., `config.Load[T]()`)
4. **Layered Error Handling**: Structured errors with HTTP status codes via `pkg/errs`
5. **Minimal Dependencies**: Only essential, well-maintained libraries

### Package Structure

```
pkg/
├── config/       # Type-safe configuration with multi-environment support
├── database/     # PostgreSQL + GORM with connection pooling
├── errs/         # Structured error types with HTTP status codes
├── http/         
│   ├── request/  # Request parsing utilities
│   ├── response/ # JSON response helpers
│   └── server/chi/ # Chi-based HTTP server
├── logger/       # Uber Zap structured logging
└── redis/        # Redis client with stats
```

## Module Reference

### 1. Config (`pkg/config`)

**Purpose**: Multi-environment configuration management with type-safe loading

**Key Features**:
- Automatic environment detection via `APP_ENV` (local, staging, production, homol)
- Base + environment-specific YAML merging
- Environment variable overrides with `APP_*` prefix
- Generic `Load[T]()` for type-safe unmarshaling
- Built on Koanf (modern alternative to Viper)

**Typical Usage**:
```go
type AppConfig struct {
    Database DatabaseConfig `koanf:"database"`
    Redis    RedisConfig    `koanf:"redis"`
}

cfg, err := config.Load[AppConfig]()
```

**FX Integration**: `config.Module` provides automatic config loading

**File Structure**:
```
config/
├── base.yaml        # Required base configuration
├── local.yaml       # Development overrides
├── staging.yaml     # Staging overrides
└── production.yaml  # Production overrides
```

### 2. Logger (`pkg/logger`)

**Purpose**: High-performance structured logging with Uber Zap

**Key Features**:
- JSON (production) and Console (development) encodings
- Log sampling for high-volume scenarios
- Typed fields for structured logging
- FX lifecycle integration (auto-sync on shutdown)

**Configuration**: Via `logger.Config` struct with sensible defaults

**Common Patterns**:
```go
log.Info("message", logger.String("key", "value"))
log.Error("failed", logger.Error(err))
```

### 3. Database (`pkg/database`)

**Purpose**: PostgreSQL connectivity with GORM ORM

**Key Features**:
- Connection pooling (configurable max connections)
- Prometheus metrics (`db_stats` gauge)
- FX lifecycle hooks (graceful shutdown)
- SSL/TLS support

**Dependencies**: `github.com/gorm.io/gorm`, `github.com/gorm.io/driver/postgres`

### 4. Redis (`pkg/redis`)

**Purpose**: Redis client with connection pooling

**Key Features**:
- Based on `github.com/redis/go-redis/v9`
- Prometheus metrics (`redis_stats` gauge)
- FX lifecycle management
- Configurable timeouts and pool sizes

### 5. HTTP Server - Chi (`pkg/http/server/chi`)

**Purpose**: Production-ready HTTP server using Chi router

**Key Features**:
- CORS middleware (github.com/go-chi/cors)
- Prometheus `/metrics` endpoint
- Health check `/health` endpoint
- Graceful shutdown with configurable timeout
- Request ID middleware
- Structured logging integration

**Response Utilities** (`pkg/http/response`):
- `JSON[T]()` - Generic type-safe JSON responses with envelope
- `Error()` - Automatic error-to-HTTP mapping via `errs.Error`
- `NoContent()` - 204 responses

**Request Utilities** (`pkg/http/request`):
- Request parsing helpers
- Located at `pkg/http/request/request.go`

### 6. Errors (`pkg/errs`)

**Purpose**: Structured HTTP-aware error type

**Structure**:
```go
type Error struct {
    Status        int      // HTTP status code
    Code          string   // Machine-readable error code
    Message       string   // Human-readable message
    Details       []Detail // Validation details (field-level)
    OriginalError error    // Wrapped original error
}
```

**Usage Pattern**: Create errors with `errs.New(code, message, status, details)`, automatically converted to HTTP responses by `response.Error()`

## Development Workflow

### Makefile Targets

| Command | Purpose |
|---------|---------|
| `make all` | Full pipeline: install tools, lint, test, coverage |
| `make install-libs` | Install dev tools (golangci-lint, govulncheck, mockery, swag, nilaway) |
| `make lint` | Run golangci-lint |
| `make vuln-check` | Security vulnerability scan |
| `make nilaway` | Nil safety analysis (Uber's static analyzer) |
| `make test` | Run all tests |
| `make cover` | Generate HTML coverage report → `reports/cover.html` |
| `make update-mocks` | Regenerate mocks with mockery |

### Dependencies/Tools

- **golangci-lint**: Linting aggregator
- **govulncheck**: Go vulnerability database scanner
- **mockery**: Mock generation for testing
- **swaggo/swag**: OpenAPI/Swagger doc generation
- **nilaway**: Uber's nil safety checker

## Testing Standards

### Test Organization

**File Naming**: Use `_test` suffix for test packages (e.g., `package mypackage_test`)

**Test Suites** (for structs with dependencies):
- Use `github.com/stretchr/testify/suite`
- Field: `sut` (System Under Test)
- Method: `SetupTest()` for initialization
- Use constructor pattern: `NewTypeName(deps...)`
- Assertions: `suite.Equal()`, `suite.True()`, etc.
- Error assertions: `suite.Require().ErrorIs()`, `suite.Require().Error()`

**Function Tests** (standalone):
- Standard `go test` functions
- Use `github.com/stretchr/testify/require` for error assertions
- Use `github.com/stretchr/testify/assert` for other assertions

### Test Structure (AAA Pattern)

```go
func (s *MyTestSuite) TestMethod_Scenario_ExpectedResult() {
    // Arrange - setup inputs and mocks
    
    // Act - execute the system under test
    
    // Assert - verify outcomes
}
```

### Mock Patterns

- **Context mocks**: Always use `mock.Anything` for `context.Context` parameters
- **No manual assertions**: Don't call `.AssertExpectations(t)` (framework handles it)

**Example**:
```go
s.repoMock.On("Update", mock.Anything, expectedData).Return(nil)
```

### Configuration in Tests

Initialize configs inline in `SetupTest()`:
```go
s.cfg = config.Config{
    App: config.App{
        BaseURL: "https://example.com",
        Name:    "Test App",
    },
    Log: config.Log{
        LogLevel: "disabled",
    },
}
```

## Key Dependencies

### Core Framework
- `go.uber.org/fx` - Dependency injection framework
- `go.uber.org/zap` - Structured logging
- `github.com/go-chi/chi/v5` - HTTP router
- `gorm.io/gorm` - ORM
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/knadh/koanf` - Configuration management

### Observability
- `github.com/prometheus/client_golang` - Metrics collection

### Testing
- `github.com/stretchr/testify` - Testing toolkit (suite, assert, require, mock)

## Code Conventions

### Error Handling
1. Use `pkg/errs.Error` for domain errors with HTTP semantics
2. Wrap errors with `fmt.Errorf("context: %w", err)` for debugging
3. Return errors explicitly; avoid panics except in fatal startup conditions

### Naming Conventions
- **Constructors**: `New()` or `NewTypeName()` for multi-package projects
- **Config structs**: `Config` suffix (e.g., `DatabaseConfig`)
- **FX modules**: Export as `Module` variable (e.g., `var Module = fx.Module(...)`)
- **Test suites**: `TypeNameTestSuite` struct

### Options Pattern
Provide functional options for flexibility:
```go
func WithTimeout(d time.Duration) Option {
    return func(o *options) {
        o.timeout = d
    }
}
```

Usage: `New(WithTimeout(5*time.Second), WithRetries(3))`

### FX Integration Pattern
Each package exports an `fx.Module`:
```go
var Module = fx.Module("packagename",
    fx.Provide(New),           // Constructor
    fx.Invoke(registerHooks),  // Lifecycle management
)
```

## Project-Specific Rules

### Testing Rules (from ai/rules/unit-tests.md)
1. **Identify test approach**: Suite for stateful structs, functions for pure logic
2. **Mock contexts**: Always use `mock.Anything` for `context.Context`
3. **Error assertions**: Use `suite.Require()` in suites, `require` in functions
4. **Package naming**: Use `_test` suffix consistently
5. **No manual expectation checks**: Framework handles `.AssertExpectations()`

### Configuration Rules
1. **Environment detection**: Set `APP_ENV=production|staging|local|homol`
2. **Override precedence**: Environment vars (`APP_*`) > env config > base config
3. **Required file**: `config/base.yaml` must exist
4. **Type safety**: Always use generics `Load[T]()` over manual unmarshaling

### HTTP Response Rules
1. **Enveloping**: Use `response.JSON[T]()` for automatic envelope wrapping
2. **Error mapping**: Let `response.Error()` convert `errs.Error` to JSON
3. **Status codes**: Set via `errs.Error.Status` field, defaults to 500

## Common Patterns

### Application Bootstrap
```go
func main() {
    fx.New(
        config.Module,
        logger.Module,
        database.Module,
        redis.Module,
        chi.Module,
        fx.Provide(NewMyService),
        fx.Invoke(registerRoutes),
    ).Run()
}
```

### Dependency Injection
```go
type Service struct {
    db     *gorm.DB
    redis  *redis.Client
    logger *zap.Logger
}

func New(db *gorm.DB, redis *redis.Client, logger *zap.Logger) *Service {
    return &Service{db: db, redis: redis, logger: logger}
}
```

### Request Handler Pattern
```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateRequest
    if err := request.DecodeJSON(r, &req); err != nil {
        response.Error(w, errs.New("invalid_request", err.Error(), 400, nil))
        return
    }
    
    result, err := h.service.Create(r.Context(), req)
    if err != nil {
        response.Error(w, err)
        return
    }
    
    response.JSON(w, http.StatusCreated, result, nil)
}
```

## Important Files

- [`go.mod`](go.mod) - Dependencies and Go version
- [`Makefile`](Makefile) - Development commands
- [`README.md`](README.md) - User-facing documentation
- [`ai/rules/unit-tests.md`](ai/rules/unit-tests.md) - Testing guidelines for AI agents
- Package READMEs: Detailed docs in each `pkg/*/README.md`

## When Adding New Modules

1. Create package under `pkg/`
2. Implement `Config` struct
3. Provide `New()` constructor accepting dependencies
4. Export `fx.Module` with providers and lifecycle hooks
5. Add Prometheus metrics (if applicable)
6. Write comprehensive `README.md` with examples
7. Add tests following suite pattern (if stateful) or function tests
8. Update main `README.md` with module reference

## Metrics & Observability

- **Database**: Pool stats exposed via `db_stats` Prometheus gauge
- **Redis**: Connection stats via `redis_stats` gauge  
- **HTTP Server**: Automatic `/metrics` endpoint (Prometheus format)
- **Logging**: Structured JSON logs in production, console in development

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `APP_ENV` | Environment selector (local/staging/production/homol) | `local` |
| `APP_*` | Override any config key (e.g., `APP_DATABASE_HOST`) | - |

## Additional Context

- **Why Koanf over Viper**: Lighter, faster, better generics support, modular
- **Why Zap over others**: Fastest structured logger, production-proven at Uber
- **Why Chi over others**: Lightweight, idiomatic, composable middleware
- **No global state**: Everything injected via FX, easier testing and composition
- **Graceful degradation**: All components handle shutdown signals cleanly

---

**Last Updated**: February 7, 2026  
**For Questions**: Check package-specific READMEs or source code comments
