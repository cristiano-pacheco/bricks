# SQLite migration execution

Read the [shared contribution contract](usage.md) when assembling filesystems or application-owned Fx wiring. This branch needs no PostgreSQL guide.

## Input and ownership

`runner.UpSQLite(db)` and `runner.DownSQLite(db)` accept an open, caller-owned `*sql.DB`. They do not accept a DSN, open a connection, or close the supplied database. Any compatible SQLite driver can supply the handle. With the repository's `github.com/mattn/go-sqlite3` driver, compilation requires CGO and a C toolchain. Standard-library `sql.Open` alone does not inherit Bricks SQLite defaults. Configure foreign keys, connection limits, and connection-local settings explicitly.

If a GORM connection already exists, obtain its handle through `gormDB.DB()`. Ownership stays with the application's original connection lifecycle. For an independently owned standard-library connection and explicit development rollback, load the [complete SQLite example](../../examples/migration-sqlite/README.md).

```go
// Fragment. The caller owns db and files.
runner := migration.NewRunner([]migration.FileSystem{files})
if err := runner.UpSQLite(db); err != nil {
    return fmt.Errorf("apply SQLite schema: %w", err)
}
// db remains open on success and failure.
```

There is no context parameter. Transactions and SQL calls use `context.Background()`. An Fx startup timeout does not cancel these operations. Arrange execution before schema-dependent consumers and serialize calls against the same database.

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
