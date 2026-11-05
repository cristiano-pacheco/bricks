# PostgreSQL migration command

Load this example when a complete Fx migration command is needed. The [shared guide](../../references/migration/usage.md) owns contribution rules; the [PostgreSQL branch](../../references/migration/postgresql.md) owns backend behavior.

The complete source is [main.go](main.go), with [up SQL](migrations/000001_create_accounts.up.sql) and [down SQL](migrations/000001_create_accounts.down.sql). It registers an application-defined filesystem group, constructs the runner, and explicitly runs `Up` in an Fx startup hook. It stops the application after startup rather than serving traffic. The runner opens and closes its own PostgreSQL connection. No Bricks database constructor, configuration module, logger interface, or GORM provider is required.

Use a disposable PostgreSQL database. Copy the whole directory into a consumer module that already selects Bricks and Fx, or run from the Bricks repository root:

```sh
export MIGRATION_DSN='postgres://user:password@localhost:5432/disposable?sslmode=disable'
go run ./skills/bricks/examples/migration-postgresql
```

In a consumer, replace the command's package path with the copied directory. The server and credentials must already exist. The example does not create or remove the database, retrieve dependencies, or start containers. Use deployment-appropriate TLS outside local development. Startup returns migration failures and exits nonzero. The startup context is not a migration cancellation mechanism.

The first run should create `migration_example_accounts`; another run should return nil without recreating it. The down file illustrates a complete pair, but `Runner` has no public PostgreSQL rollback method. Do not treat Fx stop as rollback. To use direct construction instead, call `migration.NewRunner` with the wrapped filesystem and then `Up(dsn)` without Fx.

This source has not been compiled, graph-validated, or executed. Those checks were waived for this documentation ticket. Expected behavior is source-derived, not a recorded runtime result.
