# Bricks

**Bricks** is a framework of reusable components for Go, providing ready-to-use building blocks with native Uber FX support.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks@latest
```

## Available Modules

### Config

Configuration management with YAML files, environment variable overrides, and Uber FX support.

- **Location**: `pkg/config`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/config`
- **Documentation**: [pkg/config/README.md](pkg/config/README.md)

### Database

PostgreSQL database connection module with GORM and Uber FX integration.

- **Location**: `pkg/database`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/database`
- **Documentation**: [pkg/database/README.md](pkg/database/README.md)

### Errors

Structured error handling with HTTP status codes.

- **Location**: `pkg/errs`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/errs`

### HTTP Request

High-performance JSON request parser with built-in security features for Go HTTP handlers.

- **Location**: `pkg/http/request`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/request`
- **Documentation**: [pkg/http/request/README.md](pkg/http/request/README.md)

### HTTP Response

Optimized JSON response helpers for Go HTTP handlers with high performance and minimal allocations.

- **Location**: `pkg/http/response`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/response`
- **Documentation**: [pkg/http/response/README.md](pkg/http/response/README.md)

### HTTP Server - Chi

Robust HTTP server implementation using Chi router with CORS and Uber FX support.

- **Location**: `pkg/http/server/chi`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/server/chi`
- **Documentation**: [pkg/http/server/chi/README.md](pkg/http/server/chi/README.md)

### Integration Test Kit

Integration test infrastructure for Docker containers (PostgreSQL and Redis) with automatic cleanup.

- **Location**: `pkg/itestkit`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/itestkit`
- **Documentation**: [pkg/itestkit/README.md](pkg/itestkit/README.md)

### Logger

Structured logging with slog and Uber FX integration.

- **Location**: `pkg/logger`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/logger`
- **Documentation**: [pkg/logger/README.md](pkg/logger/README.md)

### Metrics

Prometheus-based metrics collection for use case execution tracking with Uber FX integration.

- **Location**: `pkg/metrics`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/metrics`
- **Documentation**: [pkg/metrics/README.md](pkg/metrics/README.md)

### OpenTelemetry Trace

Simple and powerful OpenTelemetry tracing integration for Go applications.

- **Location**: `pkg/otel/trace`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/otel/trace`
- **Documentation**: [pkg/otel/trace/README.md](pkg/otel/trace/README.md)

### Paginator

Pagination utilities for handling pagination parameters in HTTP requests with validation and metadata generation.

- **Location**: `pkg/paginator`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/paginator`
- **Documentation**: [pkg/paginator/README.md](pkg/paginator/README.md)

### Redis

Redis client with connection pooling and Uber FX support.

- **Location**: `pkg/redis`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/redis`
- **Documentation**: [pkg/redis/README.md](pkg/redis/README.md)

### Use Case Decorator

Decorator pattern for use cases providing automatic logging, metrics, tracing, and error translation with Uber FX integration.

- **Location**: `pkg/ucdecorator`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/ucdecorator`
- **Documentation**: [pkg/ucdecorator/README.md](pkg/ucdecorator/README.md)

### Validator

Struct validation with go-playground/validator and Uber FX integration.

- **Location**: `pkg/validator`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/validator`
- **Documentation**: [pkg/validator/README.md](pkg/validator/README.md)

## License

MIT
