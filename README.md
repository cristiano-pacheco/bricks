# Bricks

**Bricks** is a framework of reusable components for Go, providing ready-to-use building blocks with native Uber FX support.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks@latest
```

## Available Modules

### Database

PostgreSQL database connection module with GORM and Uber FX integration.

- **Location**: `pkg/database`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/database`
- **Documentation**: [pkg/database/README.md](pkg/database/README.md)

### HTTP Server - Chi

Robust HTTP server implementation using Chi router with CORS and Uber FX support.

- **Location**: `pkg/http/server/chi`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/server/chi`
- **Documentation**: [pkg/http/server/chi/README.md](pkg/http/server/chi/README.md)

### HTTP Request

High-performance JSON request parser with built-in security features.

- **Location**: `pkg/http/request`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/request`
- **Features**: Content-Type validation, configurable body size limits, detailed error handling

### HTTP Response

Optimized JSON response helpers for standard HTTP handlers.

- **Location**: `pkg/http/response`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/response`
- **Features**: JSON encoding with/without envelope, error handling, high performance

### Logger

Structured logging with slog and Uber FX integration.

- **Location**: `pkg/logger`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/logger`
- **Documentation**: [pkg/logger/README.md](pkg/logger/README.md)

### Redis

Redis client with connection pooling and Uber FX support.

- **Location**: `pkg/redis`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/redis`
- **Documentation**: [pkg/redis/README.md](pkg/redis/README.md)

### Errors

Structured error handling with HTTP status codes.

- **Location**: `pkg/errs`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/errs`

## License

MIT
