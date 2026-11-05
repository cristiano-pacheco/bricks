# PostgreSQL authoring evidence

Ticket 06 documents PostgreSQL connections and migration startup against revision `51b78e7016eba26cd6c3e787519a1f15a5ceccd3`. This is maintainer evidence, not a consumer prerequisite.

## Acceptance evidence

| Ticket criterion | Delivered content and source evidence |
| --- | --- |
| Construction, typed settings, actual pool behavior, custom GORM settings, retries, errors, and ownership | The [database guide](../references/database/usage.md) covers `Module`, `NewWithLifecycle`, `New`, every `Config` field and method, validation sentinels, `ConnectionError`, and the `ConnectionStats` data type. Inspected `pkg/database/config.go`, `database.go`, `errs.go`, `fx.go`, and `stats.go`. The guide distinguishes fixed lifetime/idle ages, positive-only pool setters, ineffective `PreferSimpleProtol`, retry-wait context versus driver timeout, and GORM logger versus Bricks logger. |
| Application versus migration inputs | The database guide compares the keyword `DSN` and narrower, unescaped `PostgresDSN`. The [startup recipe](../recipes/postgresql-migrations.md) distinguishes the GORM pool from the migration driver's separate connection. Its example builds an escaped migration URL with explicit corresponding connection settings, without claiming that pool or GORM options apply to migrations. Inspected `pkg/migration/runner.go` and the existing migration references. |
| Complete Fx example and direct alternative | The [connection example](../examples/postgresql/README.md) supplies `database.Module`, a consuming invoke, YAML, an application query hook, bounded start/stop contexts, and joined errors. The database guide includes a labeled direct-construction fragment with explicit pool closure and explains application-owned providers around `NewWithLifecycle`. Complete source is supplied but compilation and execution are waived. |
| Separately loadable migration-startup composition | The recipe and [complete startup example](../examples/postgresql-startup/README.md) supply embedded up/down SQL, `fs.Sub`, an application-defined Fx filesystem group, runner construction, a dependency token ordering hook registration, and an application query after migration. Migration errors propagate through `app.Start`, which prevents later hooks from running. Database and runner connection ownership remain separate. No built-in migration module or context cancellation is invented. |
| Conditional integration disclosure | Both the database guide and startup recipe name exact planned paths for pool metrics, GORM tracing, and named multiple-database wiring. These load only for explicit integration tasks after ticket 14 authors them. No empty guides or broken Markdown links were added for those future destinations. Inspected `pkg/database/pool_metrics_fx.go` and `pkg/otel/trace/gorm_fx.go` for scope. |
| Shared writing contract and maintenance evidence | Loaded `writing-for-agents`, its skill mechanics reference, and `unslop` before Markdown authoring. Applied their disclosure, ownership, completion, and plain-prose rules during the authoring pass. Updated coverage evidence, runnable example paths and Fx classifications, inventory navigation, three routing cases, and the measurements below. Automated link and structural checks were waived. |
| Controlled PostgreSQL integration and compilation | Waived by the user, not implemented or passed. No constructor, graph, query, migration readiness, failure, repeated-run, or closure behavior was executed. Complete examples remain planned in coverage. |

`CODING_STANDARDS.md` was absent from the checkout and checked ancestors. Standards files in sibling projects do not govern this repository. Available repository instructions and the shared authoring contract applied. Only skill contents changed in the repository. No framework implementation, dependency declarations, existing package READMEs, root overview, or glossary changed.

## Verification and waivers

The user explicitly requested no reviews, lint, tests, or other checks for this skill-only task. These gates are waived for ticket resolution, not passed:

- Compilation of both complete examples and validation of their actual Fx graphs.
- Controlled PostgreSQL integration for connection, migration readiness, query execution, repeated startup, invalid configuration, unreachable database, migration failure, and pool shutdown.
- `make check-bricks-skill`, structural/link checks, final documentation acceptance, and consumer-only LLM routing evaluation.
- Repository unit/integration suites, lint, static checks, and the default suite.
- `ai-code-review`, which was not invoked.

Only source inspection, the writing pass, and size-only file measurements ran. No tests were added. `coverage.json` leaves branch/example status `planned` and execution checks empty. Later README removal remains blocked on real compilation and runtime evidence. Router construction and behavioral routing evaluation remain tickets 16 and 17. Planned integration recipes remain ticket 14 work, not unresolved implementation in this ticket.

## Token measurements

The method is UTF-8 byte length divided by four, rounded up per file, calculated with Python file-size reads. No test or acceptance checker ran for these measurements.

| Document | Bytes | Estimated tokens |
| --- | --- | --- |
| `references/database/usage.md` | 8231 | 2058 |
| `recipes/postgresql-migrations.md` | 5873 | 1469 |
| `examples/postgresql/README.md` | 3244 | 811 |
| `examples/postgresql/main.go` | 1060 | 265 |
| Connection YAML | 302 | 76 |
| `examples/postgresql-startup/README.md` | 5261 | 1316 |
| `examples/postgresql-startup/main.go` | 3766 | 942 |
| Startup YAML | 344 | 86 |
| Startup up SQL | 95 | 24 |
| Startup down SQL | 39 | 10 |

Basic connection guidance costs 2,058 estimated tokens. It exceeds the 2,000-token focused-reference target by 58. The authoring pass reduced optional integration material to conditional pointers. The remaining construction, DSN, pool, and error caveats share the same connection task, so they stay together rather than forcing an extra basic-usage read.

The migration recipe costs 1,469 tokens alone. A multi-goal connection-and-startup task reads each once for 3,527 tokens, below the 4,000-token routine-load target before the router exists. Conditional shared migration rules add 1,140 tokens; PostgreSQL backend semantics add 1,101 only when those questions arise. Broad requests for all connection and migration contracts therefore exceed the routine soft target, rather than omitting backend limitations.

The complete connection example adds 1,152 tokens. The complete startup example adds 2,378. Load these only when a full program is requested. Router-inclusive totals remain unavailable until ticket 16 supplies the router. Maintainer acceptance records are excluded from ordinary consumer loads.

## Routing evidence

`postgres-connection` selects only the database guide for basic configuration and connection questions. Configuration overlays, a complete example, telemetry, and simultaneous databases are explicit conditional branches. Imports alone do not trigger them.

`postgres-migration-startup` selects the startup recipe. Its connection, shared migration, backend migration, and complete program follow-ups have separate task conditions. `multigoal-postgres-startup` selects the database guide and recipe, deduplicated, without automatically loading telemetry or full programs.

These are expected reference selections in `routing.json`, not observed LLM runs. No evaluation model or transcript is claimed.
