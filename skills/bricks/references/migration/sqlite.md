# Apply SQLite migrations

This guide opens a SQLite file, applies embedded SQL, and closes the handle. The migration runner receives a caller-owned `*sql.DB` and never closes it itself.

## 1. Add migration files

Create `migrations/000001_create_accounts.up.sql` beside `main.go`:

```sql
CREATE TABLE migration_example_accounts (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
```

Create `migrations/000001_create_accounts.down.sql`:

```sql
DROP TABLE migration_example_accounts;
```

Use a disposable database for the example. Keep explicit `BEGIN` and `COMMIT` out of these files because the runner owns the transaction.

## 2. Open the database and apply the files

This complete `main.go` owns the connection and preserves close errors:

```go
package main

import (
    "database/sql"
    "embed"
    "errors"
    "fmt"
    "io/fs"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/migration"
    _ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var embedded embed.FS

func run() (err error) {
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return err
    }
    db, err := sql.Open("sqlite3", "file:migration-example.db?_foreign_keys=on&_busy_timeout=5000")
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, db.Close()) }()
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)
    if err := db.Ping(); err != nil {
        return err
    }
    runner := migration.NewRunner([]migration.FileSystem{migration.New(root)})
    return runner.UpSQLite(db)
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

The DSN enables foreign keys and a 5,000-millisecond busy timeout. The pool uses one open and one idle connection. `sql.Open` does not inherit Bricks SQLite constructor defaults, so these settings are explicit.

## 3. Run and repeat

With the selected Bricks and `github.com/mattn/go-sqlite3` dependencies already installed:

```sh
CGO_ENABLED=1 go run .
CGO_ENABLED=1 go run .
```

This driver requires CGO and a C toolchain. The first successful run creates `migration_example_accounts` and records version 1 in `schema_migrations`. The second successful run has no pending SQL. Both leave `migration-example.db` on disk and close the connection.

The runner has no YAML configuration type or migration defaults to copy. This example supplies the SQLite DSN in Go. For a complete command with a `-dsn` flag and an explicit development-only `-down` operation, use the [SQLite example](../../examples/migration-sqlite/README.md).

## Alternative: use an existing application connection

For a complete Bricks SQLite module setup with migration-before-query ordering and an explicit rollback command, use the [SQLite startup recipe](../../recipes/sqlite-migrations.md). For connection fields, memory databases, and direct versus Fx ownership, use the [SQLite connection guide](../database/sqlite/usage.md).

Call `gormDB.DB()` to obtain the `*sql.DB` from GORM, then pass it to `runner.UpSQLite`. Keep closure with the application's original lifecycle. `UpSQLite` and `DownSQLite` accept an open handle, not a DSN. Any compatible SQLite driver can provide it.

Neither operation accepts a context. Transactions and SQL use `context.Background()`, so an Fx startup timeout does not cancel them. Run migrations before schema-dependent consumers and serialize calls against the same database. For collecting files from multiple modules or registering Fx groups, read the [shared guide](usage.md).

## Contribution validation

SQLite validates every non-directory root entry before starting a transaction, including down files during an up run. Malformed filenames and nil filesystems produce `ErrInvalidMigration`. Every version/direction pair must be unique across all contributions, even when names or contents are identical. Both exact filename collisions and differently named duplicates fail, rather than overwriting. Filesystem order does not select a winner. File access errors retain their filesystem cause, not necessarily `ErrInvalidMigration`.

The runner sorts by numeric version, then direction and filename. Up execution selects only up files with versions greater than the current version. A database with no current row applies all available up files. A current database is a successful no-op. An empty contribution list can create the metadata table successfully without applying SQL.

## Version state and transactions

Both operations read files first, then run schema SQL and metadata changes in one `sql.Tx`. The runner creates `schema_migrations` with `version` and `dirty` columns and a unique version index. It expects at most one current row. Each successful up file replaces that row with its version and `dirty=false`. Do not modify the metadata table in migration SQL.

A pending-file failure rolls back the whole pending batch and its version updates, retaining previously committed state. The runner rolls back on panic and re-panics. A rollback failure joins the original error; a commit failure is returned. SQL must be valid inside this transaction. Do not add `BEGIN` or `COMMIT` to files or rely on operations forbidden inside a transaction.

Current dirty state fails with `ErrDatabaseDirty`. A current version greater than the highest available up version fails with `ErrDatabaseNewer`. This is a numeric bound, not a complete history check. A current version below that maximum need not have a matching file. The runner does not store checksums or detect edits to applied files. Keep the full migration history available, especially for rollback.

Malformed state can fail before the explicit checks. Multiple readable current rows or a version that overflows Go's `uint` produce `ErrInvalidSchemaVersion`. SQL scan failures for negative, null, or nonnumeric versions retain scan errors instead. Existing metadata constraints can also cause SQL errors before row validation. There is no repair or force-version API.

## Explicit development rollback

`DownSQLite(db)` rolls back all available up versions at or below the recorded current version, not one step or a requested target. It requires a matching down file for every selected up version. Missing matches produce `ErrMissingDownMigration` before down SQL executes. Unmatched down-only versions are ignored for execution, but their filenames are still validated.

The runner applies matched down files in descending version order within one transaction. It updates the version between files, deletes the current row after the last down file, and leaves the metadata table in place. Failure rolls back the whole down batch. A database without a current row returns nil after ensuring the metadata table exists. A recorded version without matching up files does not reconstruct missing history, so retain all applied files.

Call rollback only as an explicit development command against disposable data. It is never triggered by `UpSQLite`, version mismatch, Fx stop, or application shutdown.

## Failure handling

Use `errors.Is` against the exported sentinels:

| Sentinel | Meaning |
| --- | --- |
| `ErrNilDatabase` | The handle is nil. Returned before reading files. |
| `ErrInvalidMigration` | A filesystem is nil, a filename is malformed, or a version/direction pair repeats. |
| `ErrDatabaseDirty` | The current row records unresolved dirty state. |
| `ErrDatabaseNewer` | The database version exceeds the maximum available up version. |
| `ErrInvalidSchemaVersion` | Explicit row-count or Go integer overflow validation failed. |
| `ErrMissingDownMigration` | A selected up version has no matching down file. |

Loading errors are wrapped with `load sqlite migrations`; transaction and execution failures with `run sqlite up migrations` or `run sqlite down migrations`. SQL execution errors include direction, version, filename, and the driver cause. Propagate them to the caller rather than continuing startup. Not every failure is a sentinel; file access, begin, query, scan, execution, version-write, commit, and rollback errors also matter.
