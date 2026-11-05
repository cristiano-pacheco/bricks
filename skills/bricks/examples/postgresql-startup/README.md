# PostgreSQL migration startup example

This complete Fx application opens the application database, runs embedded migrations on a separate connection, queries the migrated table, and stops. Its readiness token establishes migration-hook registration before the application query hook. No queries run in constructors.

Compilation, graph validation, service-backed execution, failure checks, and shutdown checks were waived by the user for ticket 06. This is complete source, not a verified runtime example. Do not use it as evidence for removing the database README.

## Consumer setup

Copy [main.go](main.go), [config/base.yaml](config/base.yaml), and the [up](migrations/000001_startup_items.up.sql) and [down](migrations/000001_startup_items.down.sql) SQL files with their directory structure intact into a Go application that already depends on Bricks. No framework checkout is needed at runtime. Keep the application's selected dependency version and toolchain.

Provide a disposable PostgreSQL database named `bricks_skill`, a matching user, and localhost port 5432. The user must be able to create tables, including the migration-state table. Use a fresh database for this example rather than mixing its version 1 with another application's migration history. The [basic connection example](../postgresql/README.md) supplies optional local Docker commands only if container setup is needed.

From the copied example directory, run these operator commands:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
go run .
```

The sample disables TLS for local use. Configure deployment-appropriate TLS for a remote database. The keyword application DSN does not escape values; choose sample credentials that it can represent. The migration URL builder escapes its values but cannot fix limitations of the application's constructor. Never print the password or DSNs.

## Composition

`database.Module` loads typed configuration from `app.database` and provides a lifecycle-managed `*gorm.DB`. `migrationFiles` exposes the embedded SQL through `fs.Sub` and contributes a `migration.FileSystem` to the application-defined `application_migrations` group. `provideRunner` collects the group. The example has one contributor; more contributors must use distinct filenames and versions because group order is unspecified.

`registerMigrations` consumes the database to force its close-hook registration, constructs an escaped migration URL from the same configuration, and appends `runner.Up` as an `OnStart` hook. It returns `*schemaReady`. `registerApplication` consumes that token before appending its table-count query hook. Successful startup therefore requires schema application before the query. This is application-owned wiring, not automatic Bricks decoration or a built-in migration module.

The URL carries SSL and certificate settings, normalized time zone, application name, search path, and positive connection/statement/lock/idle-transaction timeouts. Pool sizing and GORM options apply only to the separate application connection. The runner closes its own migration connection after execution and discards deferred close errors. `database.Module` closes the application pool at shutdown.

The program exits zero after starting and stopping. Repeating it should find the schema current and query the same table. It is a one-shot readiness demonstration, not a server. The down file records the reverse SQL, but Bricks exposes no PostgreSQL rollback operation.

## Failure expectations and pending verification

These expected outcomes come from source inspection and still need execution against controlled PostgreSQL:

- A valid configuration and fresh database apply the table migration before the count query. A second run treats no pending migrations as success.
- An unreachable database or invalid credentials fail eager database construction before startup. Connection retry limits do not provide a hard constructor deadline.
- Invalid SQL or insufficient DDL privileges make `runner.Up` fail, prevent the application query hook from starting, and make the program exit nonzero. Earlier successful PostgreSQL versions can remain applied. Inspect dirty state before retrying.
- A query failure after migration startup fails application startup. `run` still calls `Stop` with a fresh context and preserves start and stop errors.
- Normal shutdown closes the application pool. Startup failure rolls back previously started hooks. No code path deliberately closes that pool from the migration hook.

The start context is not propagated into `runner.Up`. Do not interpret its 30-second deadline as cancellation of migration SQL or advisory locking. Source and driver errors may include connection details; handle diagnostics according to the application's secret-redaction policy.

For ordering, ownership, and operational limits, read the [startup recipe](../../recipes/postgresql-migrations.md). For migration contribution or backend details, load the [shared contract](../../references/migration/usage.md) or [PostgreSQL branch](../../references/migration/postgresql.md) only for the behavior in question. Compilation, startup, failure, repeated-run, and closure assertions remain pending, not passed.
