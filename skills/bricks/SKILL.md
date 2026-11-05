---
name: bricks
description: Use Bricks Go packages to build applications, configure components, and wire integrations with Fx or direct constructors. Use when answering Bricks usage questions or investigating its behavior in an application.
---

# Use Bricks

## Select and answer

1. Match the task and supplied constraints to the routes below. Choose the smallest sufficient set. Prefer a composition recipe for a composition task. An imported dependency does not require its complete guide.
2. Ask one focused question only when missing information changes the selection, such as Redis topology or the meaning of "add metrics". For multiple goals, combine and deduplicate the selected references. Follow additional pointers only when their stated condition applies.
3. Read the selected guide and answer with its numbered working path, including prerequisites, filenames, configuration, coherent Go code, run commands, expected results, and cleanup. Keep required wiring together. Prefer Fx unless direct construction is requested. Propagate ordinary errors, finish cleanup before reporting failure, and exit nonzero only at `main`. Use returned errors rather than `panic`, `Must` helpers, or `log.Fatal` for ordinary failures. Treat panic-capable APIs as caveats.
4. Use the inline setup in the selected guide. Read linked YAML, locale, or SQL assets only when that setup needs them. Standalone Go example programs are not distributed. Paths below are relative to this skill directory. Ordinary usage needs only these contents, not a Bricks checkout. State what was actually executed and what remains unverified.

## Application components

- Load YAML, select a subtree, apply overlays, or resolve environment references with [configuration](references/config/usage.md).
- Configure logging, options, child loggers, backend access, or flushing with [logging](references/logger/usage.md).
- Return coded errors, HTTP statuses, details, or original errors with [structured errors](references/errs/usage.md).
- Check structs, scalars, aliases, custom rules, or validation translations with [validation](references/validator/usage.md). JSON decoding alone is not validation.
- Parse page parameters, cap sizes, compute offsets, or build metadata with [pagination](references/paginator/usage.md). For an HTTP list endpoint, use [pagination responses](recipes/pagination-response.md) instead.

## Storage

- Connect, configure pooling, or close [PostgreSQL](references/database/usage.md) or [SQLite](references/database/sqlite/usage.md). For simultaneous connections and Fx naming, use [multiple databases](recipes/multiple-databases.md).
- Connect Redis using only the selected topology: [standalone](references/redis/standalone.md), [cluster](references/redis/cluster.md), [Sentinel address list](references/redis/sentinel.md), or [failover URL](references/redis/failover.md). Do not load other topology guides for comparison unless requested. When Sentinel discovery is requested without a specific API branch, use the Sentinel guide.
- Tune Redis fields, defaults, TLS, or retries with [Redis configuration](references/redis/configuration.md). For wrapper versus universal-client providers, namespaces, callbacks, options, or statistics, use [Redis client behavior](references/redis/client.md).
- Compose migration files, handle version collisions, or understand rollback with [migration contracts](references/migration/usage.md). For a backend connection alone, select [PostgreSQL migration inputs](references/migration/postgresql.md) or [SQLite migration inputs](references/migration/sqlite.md).
- Run migrations during application startup with [PostgreSQL migrations](recipes/postgresql-migrations.md) or [SQLite migrations](recipes/sqlite-migrations.md). Select the requested backend.

## HTTP

- Decode JSON, set body limits, or understand content types and parse errors with [JSON requests](references/http/request/usage.md). Extract Chi path values with [path parameters](references/http/server/chi/request/usage.md).
- Return enveloped, raw, empty, or error responses with [HTTP responses](references/http/response/usage.md).
- Connect decoding, use-case validation, execution, decoration, and responses with [complete HTTP flow](recipes/http-flow.md). This recipe owns its required wiring; load component guides only for deeper options.
- Register Chi routes, bind Fx route groups, or own server startup and shutdown with [Chi setup](references/http/server/chi/usage.md).
- Configure Chi fields and deployment scenarios with [server YAML](references/http/server/chi/configuration.md). Load only the requested optional branch for [middleware](references/http/server/chi/middleware.md), [CORS](references/http/server/chi/cors.md), [Swagger](references/http/server/chi/swagger.md), or [server logging](references/http/server/chi/logging.md).

## Use cases and translation

- Wrap a use case explicitly without factory configuration with [manual chains](references/ucdecorator/manual.md).
- Configure a factory, supply its dependencies, choose flags and names, or explicitly wrap a use case with [decorator factories](references/ucdecorator/factory.md).
- Add a specific wrapper integration with [decorator integrations](references/ucdecorator/integrations.md). For logging, metrics, and exported traces together, use [decorator observability](recipes/decorator-observability.md).
- Load locale files, interpolate messages, or select language and fallback with [translation](references/i18n/usage.md). For structured-error translation around a use case, use [localized use cases](recipes/localized-usecases.md).
- Load i18n support contracts only when extending or using them directly: [configuration type](references/i18n/config/usage.md), [translation DTO](references/i18n/dto/usage.md), [locale filesystem contribution](references/i18n/locale/usage.md), [translator interface](references/i18n/ports/usage.md), or [direct translation service](references/i18n/service/usage.md). Routine translation does not require this set.

## Metrics and tracing

- Record use-case durations and outcomes with [use-case metrics](references/metrics/usage.md).
- Expose metrics on an independent listener with [metrics server](references/http/server/metrics/usage.md), or select [Chi's metrics listener](recipes/metrics-chi.md) when that arrangement is requested.
- Connect collectors to a listener or combine global and application registries with [registry wiring](recipes/metrics-registries.md). Listener installation alone does not unify registries.
- Instrument pools with [database pool metrics](recipes/database-pool-metrics.md) or [Redis pool metrics](recipes/redis-pool-metrics.md), rather than the use-case metrics guide.
- Initialize tracing, select an exporter, create spans, or own shutdown with [tracing](references/otel/trace/usage.md). For storage instrumentation, select [GORM tracing](recipes/gorm-tracing.md) or [eligible Redis tracing](recipes/redis-tracing.md).

## Integration tests and source investigation

- Start controlled PostgreSQL test infrastructure and understand client and cleanup ownership with [PostgreSQL testkit](references/itestkit/usage.md). For Redis, select [Redis testkit](references/itestkit/redis.md). For test migrations, select [test migration setup](references/itestkit/migrations.md).
- Wire PostgreSQL and Redis into one Fx test application with [integration testing](recipes/integration-testing.md).
- Investigate local source, undocumented behavior, API discrepancies, or a mismatch with these examples with [local-source discovery](references/local-source.md). This is conditional, not a usage prerequisite. Follow its module-resolution and bounded-search steps; report missing local source instead of downloading or searching the machine.

For configuration questions, use the selected component's minimal and full-field YAML and field explanations. Keep incompatible scenarios separate. Packages documented as code-only have no invented YAML settings. The router defines no application schema.
