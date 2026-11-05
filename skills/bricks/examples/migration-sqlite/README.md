# Run SQLite migrations on a disposable database

This command opens its own standard-library SQLite connection, applies embedded SQL, and closes the handle. It uses `github.com/mattn/go-sqlite3`, not a Bricks database constructor or Fx.

## 1. Copy the command and SQL files

Copy [main.go](main.go), [the up file](migrations/000001_create_accounts.up.sql), and [the down file](migrations/000001_create_accounts.down.sql) into a consumer module that already selects Bricks and `github.com/mattn/go-sqlite3`:

```text
main.go
migrations/
  000001_create_accounts.up.sql
  000001_create_accounts.down.sql
```

Keep the directory structure so the embed directive finds both files. The driver requires CGO and a C toolchain.

## 2. Choose the disposable database

With no flags, the command uses this DSN:

```text
file:migration-example.db?_foreign_keys=on&_busy_timeout=5000
```

It creates `migration-example.db` in the process working directory, enables foreign keys, and sets a 5,000-millisecond busy timeout. The program sets one open and one idle connection, then pings before running migrations.

There is no YAML file for this command. The migration runner has no YAML configuration type. Use `-dsn` to replace the connection string. A replacement DSN must include the settings you need; it does not inherit the example's DSN options or Bricks constructor defaults.

## 3. Apply the migrations

From the copied command directory:

```sh
CGO_ENABLED=1 go run .
```

From the Bricks repository root instead:

```sh
CGO_ENABLED=1 go run ./skills/bricks/examples/migration-sqlite
```

A successful first run should create `migration_example_accounts` and record version 1. The connection closes on success or failure, and close errors join the returned error. The database file remains on disk.

## 4. Repeat the up command

Run the same command again. A current schema should be a successful no-op. The command creates no containers and retrieves no dependencies itself.

To select another disposable file from the copied directory:

```sh
CGO_ENABLED=1 go run . -dsn 'file:another-example.db?_foreign_keys=on&_busy_timeout=5000'
```

## 5. Explicitly undo all migrations during development

Only for the disposable default database:

```sh
CGO_ENABLED=1 go run . -down
```

Or from the repository root:

```sh
CGO_ENABLED=1 go run ./skills/bricks/examples/migration-sqlite -down
```

This calls `DownSQLite`, which rolls back all available applied up versions, not a single step. It should drop the example table and clear the current metadata row. It leaves the database file and `schema_migrations` table in place. A later up command should reapply the schema.

Supply the same `-dsn` used for up if you selected a different database. Keep rollback as an explicit development command. Startup and shutdown never trigger it automatically.

## How to adapt the command

The program uses `fs.Sub` to expose SQL at the filesystem root and wraps it with `migration.New`. `UpSQLite` receives the open `*sql.DB`, applies pending migrations transactionally, and leaves connection ownership with the command. Neither operation accepts a context. Do not add explicit `BEGIN` or `COMMIT` to the migration files.

For contribution and filename rules, read the [shared guide](../../references/migration/usage.md). For version checks, error sentinels, transaction behavior, and rollback limits, read the [SQLite branch](../../references/migration/sqlite.md).
