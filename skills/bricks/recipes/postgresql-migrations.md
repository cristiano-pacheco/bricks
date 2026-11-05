# PostgreSQL migrations before readiness

Use this recipe when an application needs a Bricks PostgreSQL connection and SQL migrations before its serving or worker hooks start. Installing `database.Module` alone does not migrate the schema. Bricks supplies no migration Fx module or automatic migration group.

## Ordered startup

The [complete application](../examples/postgresql-startup/README.md) includes YAML, embedded SQL, an application-owned filesystem group, runner construction, a migration readiness dependency, an application query, and shutdown. Load it when reproducing the full wiring.

1. Install `database.Module` for the unnamed `*gorm.DB` and its close hook. Its constructor connects eagerly, so the target database must already exist. This recipe migrates tables, not the PostgreSQL database itself.
2. Each contributor returns `migration.New(rootFS)` through `fx.Annotate` with `fx.ResultTags` naming the same application-defined group. The example uses `fs.Sub` so embedded files appear at the contribution root.
3. An application provider collects `[]migration.FileSystem` with an `fx.In` group field and calls `migration.NewRunner`. Give contributions unique filenames and versions. Fx value-group order is unspecified; it cannot establish precedence for conflicting contributions.
4. A provider consumes the runner, `config.Config[database.Config]`, `*gorm.DB`, and `fx.Lifecycle`. Consuming the database forces its constructor and shutdown-hook registration first. Append a migration `OnStart` hook and return an application-owned readiness token.
5. Serving or worker registration consumes that token and appends its own hook. Construction of the token registers the migration hook first. Fx runs starts in registration order and stops in reverse order. Merely importing modules or receiving a `*gorm.DB` does not establish migration readiness.

The token orders hook registration; it is not itself proof that migrations ran. Start request handling only in the later `OnStart`, after the earlier migration hook returns nil. Do not query tables in constructors. The complete example performs a startup query instead of starting a server, so it introduces no unrelated transport setup.

For contribution layouts, filenames, collision rules, and public operation selection, read the [shared migration contract](../references/migration/usage.md). For PostgreSQL version state, dirty migrations, transaction boundaries, error wrapping, and absent rollback operations, read the [PostgreSQL migration branch](../references/migration/postgresql.md). This recipe owns ordering and connection composition, not those package contracts.

## Two connection owners

`database.Module` owns the application GORM pool and closes it on normal shutdown. `runner.Up(url)` opens a separate migration connection and defers its own close after successful migrate-instance construction. It does not reuse GORM or the application pool. Do not close GORM from the migration hook. Application pool limits do not configure the migration connection.

Use a migration URL built with `net/url` and explicit deployment TLS policy. The complete example derives host, credentials, database name, SSL mode, certificates, time zone, search path, application name, and positive timeout settings from the typed configuration. Pool and GORM fields deliberately have no URL equivalent. `Config.DSN()` is the application's keyword DSN, not this recipe's migration URL. `Config.PostgresDSN()` omits certificate and timeout settings and does not escape values; it is not a lossless conversion. For those constructor contracts, read the [database guide](../references/database/usage.md) only when changing connection setup.

A deployment can instead supply a separately escaped migration URL with distinct privileges. Keep its target database and schema consistent with the application connection. Never print secrets or either DSN. Database permissions must permit the application's operations, while migration credentials must permit the required DDL and migration-state updates.

## Failure and shutdown

Configuration or database construction failures prevent startup. A migration error returns from its `OnStart` with context, causes `app.Start` to fail, and prevents later application hooks from running. The example calls `app.Stop` after any start attempt, with a fresh context, and joins errors rather than exiting before cleanup. Fx rolls back previously started hooks on startup failure. Successfully applied earlier PostgreSQL migrations may remain when a later migration fails. Startup failure is not a schema rollback.

The migration API accepts no context. An Fx start timeout does not cancel `runner.Up`, advisory locking, or SQL execution. Set suitable PostgreSQL connection and statement policies; use an externally controlled migration job when application startup needs a cancellable deployment boundary. The example supplies an Fx context deadline but does not present it as a hard migration deadline. Close errors from the runner's deferred close are discarded by its implementation.

If a serving hook starts successfully, its `OnStop` must stop accepting work before the database's close hook runs. The explicit readiness dependency keeps that application hook after migration registration. No serving hook is needed for the example's one-shot readiness query.

## Optional integrations

Only for pool monitoring, consult the planned `recipes/database-pool-metrics.md` after ticket 14 authors it. Only for tracing, consult the planned `recipes/gorm-tracing.md` after ticket 14 authors it. Only for simultaneous database connections, consult the planned `recipes/multiple-databases.md` after ticket 14 authors it. Those destinations are not yet distributed guides. Their absence does not block this startup recipe; do not load unrelated branches for basic migrations.
