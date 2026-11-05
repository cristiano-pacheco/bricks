# Apply migrations before a PostgreSQL startup query

This one-shot Fx application opens the application pool, applies embedded SQL through a separate migration connection, queries the new table, and stops. A migration failure prevents the query hook from starting.

## 1. Copy the application and schema files

Copy [main.go](main.go), [config/base.yaml](config/base.yaml), and both SQL files with this layout:

```text
main.go
config/
  base.yaml
migrations/
  000001_startup_items.up.sql
  000001_startup_items.down.sql
```

Use a Go application that already selects the desired Bricks version and toolchain. The copied program needs no framework checkout at runtime.

The [up file](migrations/000001_startup_items.up.sql) creates `bricks_skill_startup_items`. The [down file](migrations/000001_startup_items.down.sql) records the reverse SQL, but Bricks exposes no PostgreSQL rollback operation. Shutdown does not run it.

## 2. Configure both connections

The supplied YAML lists every scalar `database.Config` field. The program derives its migration URL from these same settings:

```yaml
app:
  database:
    host: localhost # Required.
    port: 5432 # Required. Range 1 to 65535.
    name: bricks_skill # Required. The database must already exist.
    user: bricks_skill # Required.
    password: env://POSTGRES_PASSWORD
    sslmode: false # Local development only.
    sslcert: "" # Optional certificate paths, used only with TLS enabled.
    sslkey: ""
    sslrootcert: ""
    maxopenconnections: 10
    maxidleconnections: 2
    connecttimeout: 5 # Seconds.
    statementtimeout: 5000 # Milliseconds.
    locktimeout: 5000 # Milliseconds.
    idleintransaction: 0 # Milliseconds. Zero leaves server policy in effect.
    timezone: UTC
    applicationname: bricks_skill_startup
    searchpath: public
    preparestmt: false
    skipdefaulttransaction: false
    disableforeignkeyconstraint: false
    enablelogs: false # Supply custom GORM Logger implementations in Go.
    prefersimpleprotol: false # Exported spelling. Currently has no effect.
```

Use a fresh disposable database rather than mixing this example's version 1 with another application's migration history. The database and user must already exist. The user needs table-creation and migration-state permissions. For optional local Docker setup, use step 3 of the [connection example](../postgresql/README.md#3-provide-a-disposable-postgresql-database).

Set deployment-appropriate TLS for remote databases. `Logger` and `NamingStrategy` are Go-only interfaces. For field defaults and behavior, read the [connection configuration reference](../../references/database/usage.md#configuration-reference).

The application constructor builds a keyword DSN without escaping values. Choose credentials that it can represent. The migration URL builder escapes values but cannot fix that application-connection limitation. Never print passwords or either DSN.

## 3. Run the application

From the copied directory:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
go run .
```

Successful startup applies the table migration before the table-count query. The program then shuts down and exits zero. It is not a long-running server.

## 4. Repeat the command

```sh
go run .
```

With a current schema, migration execution should return nil and the table-count query should succeed again. Use the same configuration and migration files when repeating the command.

## How the startup order works

1. `database.Module` loads `app.database` and provides a lifecycle-managed `*gorm.DB`.
2. `migrationFiles` uses `fs.Sub` to expose SQL at the filesystem root. Its result joins the application-defined `application_migrations` group.
3. `provideRunner` collects that group. Additional contributors need distinct filenames and versions because Fx group order is unspecified.
4. `registerMigrations` requests `*gorm.DB`, forcing pool construction and close-hook registration first. It builds an escaped URL and registers `runner.Up` in `OnStart`, then returns `*schemaReady`.
5. `registerApplication` requests `*schemaReady` before registering its table-count hook. The migration hook therefore starts first. No queries run in constructors.

The token orders registration; SQL has not yet run when the token is returned. Keep that dependency if you replace the query with a server or worker. A serving hook must stop accepting work before the database close hook runs.

## Connection ownership and failures

The application pool and migration connection have separate owners. The URL includes SSL and certificate settings, normalized time zone, application name, search path, and positive connection/statement/lock/idle-transaction timeouts. Pool and GORM options apply only to the application pool. The runner closes its own connection after execution and discards deferred close errors. Fx closes the application pool at shutdown.

An unreachable database or invalid credentials fail eager pool construction. Invalid SQL or insufficient DDL privileges fail the migration hook and prevent the query hook from starting. Earlier successful PostgreSQL versions can remain applied; inspect dirty state before retrying. Startup failure is not schema rollback.

A query failure also fails startup. After a start attempt, `run` calls `Stop` with a fresh context and preserves both errors. Fx rolls back previously started hooks on startup failure. No migration hook deliberately closes the application pool.

The 30-second start context does not reach `runner.Up`, its advisory locking, or its SQL calls. It is not a hard migration deadline. Handle source and driver diagnostics under your application's secret-redaction policy.

For changing hook ordering or using separate migration credentials, read the [startup recipe](../../recipes/postgresql-migrations.md). For file contribution rules, read the [shared migration guide](../../references/migration/usage.md). For dirty state or PostgreSQL transaction behavior, read the [PostgreSQL branch](../../references/migration/postgresql.md).
