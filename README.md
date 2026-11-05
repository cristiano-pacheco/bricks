# Bricks

**Bricks** is a framework of reusable components for Go, providing ready-to-use building blocks with native Uber FX support.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks@latest
```

## Usage guides

Use the [Bricks skill](skills/bricks/SKILL.md) to select a package guide or composition recipe for your task. Copy `skills/bricks/` into your application's skill directory to use it without a framework checkout. Guides include inline Go setup; `examples/` contains copyable YAML, locale files, and SQL assets rather than standalone Go programs.

Package READMEs remain available while replacement walkthrough and execution gates are unverified.

## Available modules

### Config

Configuration management with YAML files, environment variable overrides, and Uber FX support.

- **Location**: `pkg/config`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/config`
- **Documentation**: [configuration guide](skills/bricks/references/config/usage.md)

### Database

PostgreSQL database connection module with GORM and Uber FX integration.

- **Location**: `pkg/database`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/database`
- **Documentation**: [PostgreSQL guide](skills/bricks/references/database/usage.md)

### SQLite database

SQLite connection construction with GORM, `mattn/go-sqlite3`, and Uber FX lifecycle support.

- **Location**: `pkg/database/sqlite`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/database/sqlite`
- **Documentation**: [SQLite guide](skills/bricks/references/database/sqlite/usage.md)

### Errors

Structured error handling with HTTP status codes.

- **Location**: `pkg/errs`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/errs`
- **Documentation**: [structured errors guide](skills/bricks/references/errs/usage.md)

### HTTP Request

JSON request decoding with body limits and parse-error handling for Go HTTP handlers.

- **Location**: `pkg/http/request`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/request`
- **Documentation**: [JSON request guide](skills/bricks/references/http/request/usage.md)

### HTTP Response

Enveloped, raw, empty, and error responses for Go HTTP handlers.

- **Location**: `pkg/http/response`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/response`
- **Documentation**: [HTTP response guide](skills/bricks/references/http/response/usage.md)

### HTTP Server - Chi

HTTP server implementation using Chi router with CORS, configurable listeners, and Uber FX support.

- **Location**: `pkg/http/server/chi`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/http/server/chi`
- **Documentation**: [Chi setup guide](skills/bricks/references/http/server/chi/usage.md)

### i18n

Internationalization package with locale loading, translation lookups, template interpolation, and error translation with Uber FX integration.

- **Location**: `pkg/i18n`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/i18n`
- **Documentation**: [translation guide](skills/bricks/references/i18n/usage.md)

### Integration Test Kit

Integration test infrastructure for Docker containers (PostgreSQL and Redis) with automatic cleanup.

- **Location**: `pkg/itestkit`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/itestkit`
- **Documentation**: [PostgreSQL testkit guide](skills/bricks/references/itestkit/usage.md) or [Redis testkit guide](skills/bricks/references/itestkit/redis.md)

### Logger

Structured logging with slog and Uber FX integration.

- **Location**: `pkg/logger`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/logger`
- **Documentation**: [logging guide](skills/bricks/references/logger/usage.md)

### Metrics

Prometheus-based metrics collection for use case execution tracking with Uber FX integration.

- **Location**: `pkg/metrics`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/metrics`
- **Documentation**: [use-case metrics guide](skills/bricks/references/metrics/usage.md)

### Migration

Database migration package with co-located, per-module SQL migration support using embedded filesystems and `golang-migrate`, with Uber FX integration.

- **Location**: `pkg/migration`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/migration`
- **Documentation**: [migration guide](skills/bricks/references/migration/usage.md)

### OpenTelemetry Trace

Simple and powerful OpenTelemetry tracing integration for Go applications.

- **Location**: `pkg/otel/trace`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/otel/trace`
- **Documentation**: [tracing guide](skills/bricks/references/otel/trace/usage.md)

### Paginator

Pagination utilities for handling pagination parameters in HTTP requests with validation and metadata generation.

- **Location**: `pkg/paginator`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/paginator`
- **Documentation**: [pagination guide](skills/bricks/references/paginator/usage.md)

### Redis

Redis client with connection pooling and Uber FX support.

- **Location**: `pkg/redis`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/redis`
- **Documentation**: [standalone Redis guide](skills/bricks/references/redis/standalone.md). For Cluster, Sentinel, or failover URL setup, select the topology in the [skill router](skills/bricks/SKILL.md#storage).

### Use Case Decorator

Explicit use-case wrapping with logging, metrics, tracing, and error translation, through manual chains or an Fx-provided factory.

- **Location**: `pkg/ucdecorator`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/ucdecorator`
- **Documentation**: [manual chains](skills/bricks/references/ucdecorator/manual.md) or [configured factories](skills/bricks/references/ucdecorator/factory.md)

### Validator

Struct validation with go-playground/validator and Uber FX integration.

- **Location**: `pkg/validator`
- **Import**: `github.com/cristiano-pacheco/bricks/pkg/validator`
- **Documentation**: [validation guide](skills/bricks/references/validator/usage.md)

## License

MIT
