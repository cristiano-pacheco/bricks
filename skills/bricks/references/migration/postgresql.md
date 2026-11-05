# PostgreSQL migration execution

Read the [shared contribution contract](usage.md) when assembling filesystems or application-owned Fx wiring. This branch needs no SQLite guide.

## Input and ownership

Call `runner.Up(dsn)` with a PostgreSQL URL, such as `postgres://user:password@localhost:5432/app?sslmode=disable`. Use deployment-appropriate TLS settings and URL-escape credentials. Do not log a DSN containing secrets. Bricks passes this string to golang-migrate's registered PostgreSQL driver, not to the Bricks database constructor. An existing GORM or `*sql.DB` connection is neither required nor reused.

The driver opens its own database connection. After successful migrate-instance construction, `Up` defers `m.Close()` to close the source and database driver. Close errors are discarded, so a nil result is not proof that close succeeded. The method has no context argument or application-level timeout option. Configure supported connection parameters in the URL; an Fx hook's context is not propagated into `Up`. The pinned driver accepts `x-statement-timeout` in milliseconds, defaulting to zero without a statement deadline. It uses PostgreSQL advisory locking; a blocked database lock is not cancelled by the Fx hook context.

Direct execution is this fragment:

```go
runner := migration.NewRunner([]migration.FileSystem{files})
if err := runner.Up(dsn); err != nil {
    return fmt.Errorf("apply PostgreSQL schema: %w", err)
}
```

For a complete Fx command with embedded files, an environment-supplied DSN, and explicit startup execution, load the [PostgreSQL example](../../examples/migration-postgresql/README.md). It does not install a Bricks database module.

## Contributions and versions

Bricks merges flat-root files into an in-memory filesystem under `migrations/`. Identical filenames are overwritten by later entries in the `NewRunner` slice before parsing. This is not conflict detection. Different filenames with the same numeric version and direction survive the merge and are rejected by the pinned iofs source as duplicate migrations. An up/down pair at one version is valid. Malformed filenames are skipped by iofs, whereas filesystem read failures abort the merge. A nil filesystem is not validated into a Bricks sentinel; provide valid filesystems.

The pinned golang-migrate driver orders versions numerically and records migration state in `schema_migrations` by default. A current schema returns `migrate.ErrNoChange`, which Bricks converts to nil. An empty source on an unversioned database can instead fail while locating its first migration. Preserve historical files. The upstream engine requires the database's current version to exist in the source; Bricks does not apply SQLite's `ErrDatabaseNewer` check.

## Transactions, errors, and rollback limits

Bricks invokes upstream `Up`, not a transaction enclosing the whole pending batch. Golang-migrate records dirty state before each migration and clears it after success. Earlier successful versions remain applied when a later migration fails. The PostgreSQL driver sends a file as one SQL execution by default. Its URL option `x-multi-statement=true` enables semicolon-based splitting and changes execution boundaries. `x-multi-statement-max-size` sets the maximum statement size in bytes, with a 10 MiB default. Keep the default whole-file mode unless splitting is required. Write SQL transaction boundaries deliberately where needed, and account for PostgreSQL operations that cannot run inside a transaction. Do not assume SQLite's all-pending transaction behavior here.

Errors preserve causes with `%w` and identify the stage:

- `merge migration filesystems` wraps directory or file-read failures.
- `create iofs source` wraps source initialization and duplicate version/direction errors.
- `create migrate instance` wraps DSN, driver, connection, and initialization failures.
- `run migrations` wraps execution, locking, version, and dirty-state failures.

Use `errors.As` with upstream `migrate.ErrDirty` for dirty state. The exported Bricks SQLite sentinels are not a PostgreSQL error taxonomy. Resolve a failed migration through a separately controlled operational procedure after inspecting actual schema state. `Runner` exposes no PostgreSQL `Down`, steps, force-version, repair, or automatic downgrade. A `.down.sql` file does not make PostgreSQL rollback callable through this package.
