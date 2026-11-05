# Run a PostgreSQL migration command with Fx

This one-shot command collects embedded SQL through an application-owned Fx group and calls `Runner.Up` during startup. It opens its own PostgreSQL connection. It needs no Bricks database module, configuration module, logger interface, or GORM provider.

## 1. Copy the command and SQL files

Copy [main.go](main.go), [the up file](migrations/000001_create_accounts.up.sql), and [the down file](migrations/000001_create_accounts.down.sql) into a consumer module that already selects Bricks and Fx:

```text
main.go
migrations/
  000001_create_accounts.up.sql
  000001_create_accounts.down.sql
```

Keep the directories intact so the embed directive can find the SQL files. Use a fresh disposable database instead of combining this example's version 1 with existing migration history.

## 2. Supply the database URL

The PostgreSQL server, database, and credentials must already exist. Replace this local URL with your disposable database's values:

```sh
export MIGRATION_DSN='postgres://user:password@localhost:5432/disposable?sslmode=disable'
```

The command reads `MIGRATION_DSN` directly and returns an error when it is empty. It has no YAML configuration file. The migration package has no YAML configuration type or built-in configuration provider.

URL-escape credentials and use deployment-appropriate TLS outside local development. Do not print a secret-bearing URL. For an application that already stores database fields in YAML, use the [startup example](../postgresql-startup/README.md), which derives an escaped migration URL from them.

## 3. Run the command

From the copied command directory:

```sh
go run .
```

Alternatively, run the supplied source from the Bricks repository root:

```sh
go run ./skills/bricks/examples/migration-postgresql
```

A successful first run creates `migration_example_accounts` and records migration state. The command stops after startup instead of serving traffic. A migration failure returns an error and produces a nonzero exit.

## 4. Run it again

Use the same command and URL. A current schema should return nil without recreating the table. These are expected outcomes from source inspection, not recorded runtime results.

The down file illustrates a complete pair. `Runner` has no public PostgreSQL rollback operation, and Fx stop never applies down SQL. The command does not create or remove the database, retrieve dependencies, or start containers.

## How to adapt the wiring

The filesystem provider uses `fs.Sub` so the SQL files appear at its root. `fx.Annotate` adds it to the application-defined `application_migrations` group. `provideRunner` collects the group and calls `migration.NewRunner`. The invoke registers an explicit `Up` startup hook. Providing the runner alone would not execute migrations.

Give additional contributions unique filenames and versions. Fx group order does not establish precedence. For file naming and grouping rules, read the [shared guide](../../references/migration/usage.md). For transaction boundaries, dirty state, errors, and URL options, read the [PostgreSQL branch](../../references/migration/postgresql.md).

For direct construction without Fx, wrap the filesystem, call `migration.NewRunner`, then call `Up(dsn)`. No application connection is required or reused. The runner closes its own connection after successful migrate-instance construction, but discards deferred close errors. A startup context does not cancel migration SQL or advisory locking.
