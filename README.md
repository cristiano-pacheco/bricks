# Bricks

Reusable Go components for HTTP services, databases, and observability, with Uber Fx integration.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks@latest
```

## Agent skill

Install the `bricks` skill for package usage guides and integration recipes:

```bash
npx skills add cristiano-pacheco/bricks --skill=bricks
```

Run it from your application directory. Add `--global` to install it for your user instead of the current project. Requires Node.js and npm.

For manual browsing, start with the [skill index](skills/bricks/SKILL.md).

## Available modules

Import packages with the prefix `github.com/cristiano-pacheco/bricks/`.

| Package | Purpose | Guide |
| --- | --- | --- |
| `pkg/config` | YAML configuration and environment overrides | [Configuration](skills/bricks/references/config/usage.md) |
| `pkg/database` | PostgreSQL with GORM | [PostgreSQL](skills/bricks/references/database/usage.md) |
| `pkg/database/sqlite` | SQLite with GORM | [SQLite](skills/bricks/references/database/sqlite/usage.md) |
| `pkg/errs` | Structured errors and HTTP status codes | [Errors](skills/bricks/references/errs/usage.md) |
| `pkg/http/request` | JSON request decoding and body limits | [Requests](skills/bricks/references/http/request/usage.md) |
| `pkg/http/response` | JSON, empty, and error responses | [Responses](skills/bricks/references/http/response/usage.md) |
| `pkg/http/server/chi` | Chi HTTP server and CORS | [Chi](skills/bricks/references/http/server/chi/usage.md) |
| `pkg/i18n` | Translations and locale loading | [i18n](skills/bricks/references/i18n/usage.md) |
| `pkg/itestkit` | PostgreSQL and Redis containers for tests | [PostgreSQL](skills/bricks/references/itestkit/usage.md), [Redis](skills/bricks/references/itestkit/redis.md) |
| `pkg/logger` | Structured logging with slog | [Logging](skills/bricks/references/logger/usage.md) |
| `pkg/metrics` | Prometheus use-case metrics | [Metrics](skills/bricks/references/metrics/usage.md) |
| `pkg/migration` | Embedded SQL migrations per module | [Migrations](skills/bricks/references/migration/usage.md) |
| `pkg/otel/trace` | OpenTelemetry tracing | [Tracing](skills/bricks/references/otel/trace/usage.md) |
| `pkg/paginator` | Pagination parameters and metadata | [Pagination](skills/bricks/references/paginator/usage.md) |
| `pkg/redis` | Redis connections and pooling | [Topology selection](skills/bricks/SKILL.md#storage) |
| `pkg/ucdecorator` | Use-case logging, metrics, tracing, and error translation | [Manual chains](skills/bricks/references/ucdecorator/manual.md), [factories](skills/bricks/references/ucdecorator/factory.md) |
| `pkg/validator` | Struct validation | [Validation](skills/bricks/references/validator/usage.md) |

## License

[MIT](LICENSE)
