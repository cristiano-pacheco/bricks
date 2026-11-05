# SQLite migration command

Load this example when a complete SQLite migration command with caller-owned connections or explicit development rollback is needed. The [shared guide](../../references/migration/usage.md) owns contribution rules; the [SQLite branch](../../references/migration/sqlite.md) owns backend behavior.

The complete source is [main.go](main.go), with [up SQL](migrations/000001_create_accounts.up.sql) and [down SQL](migrations/000001_create_accounts.down.sql). It uses standard-library `sql.Open` and the registered `go-sqlite3` driver, not a Bricks database constructor. It sets one open and one idle connection, pings the database, runs the selected public migration operation, and closes its owned handle even on failure. Close errors join the returned error.

Copy the entire directory into a consumer module that already selects Bricks and `github.com/mattn/go-sqlite3`, or run from the repository root. CGO and a C toolchain are required for this driver:

```sh
CGO_ENABLED=1 go run ./skills/bricks/examples/migration-sqlite
CGO_ENABLED=1 go run ./skills/bricks/examples/migration-sqlite
```

The default DSN creates `migration-example.db` in the working directory, enables foreign keys, and sets a 5,000-millisecond busy timeout. The first run should create `migration_example_accounts` and record version 1; the second should be a no-op. Use `-dsn` for a different disposable database. Supplying a DSN replaces these connection settings rather than inheriting Bricks defaults.

To explicitly undo all migrations on this disposable database:

```sh
CGO_ENABLED=1 go run ./skills/bricks/examples/migration-sqlite -down
```

The down command should drop the example table and clear the metadata row. It leaves the database file and metadata table in place. A later up run should reapply the migration. In a consumer, replace the commands' package paths with the copied directory. The example creates no containers and retrieves no dependencies itself. It is a direct-construction example, not an Fx or application database-lifecycle composition.

This source has not been compiled or executed. Those checks were waived for this documentation ticket. Expected behavior is source-derived, not a recorded runtime result.
