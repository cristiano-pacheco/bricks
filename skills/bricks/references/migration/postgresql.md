# Apply PostgreSQL migrations

This command applies embedded SQL to an existing PostgreSQL database. It opens its own migration connection and does not need GORM or a Bricks database provider.

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

Use a fresh database for this example. In an existing application, choose an unused numeric version instead of reusing version 1.

## 2. Create the migration command

This complete direct-construction `main.go` embeds the SQL directory and returns migration failures:

```go
package main

import (
    "embed"
    "errors"
    "fmt"
    "io/fs"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/migration"
)

//go:embed migrations/*.sql
var embedded embed.FS

func run() error {
    dsn := os.Getenv("MIGRATION_DSN")
    if dsn == "" {
        return errors.New("MIGRATION_DSN is required")
    }
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return err
    }
    runner := migration.NewRunner([]migration.FileSystem{migration.New(root)})
    return runner.Up(dsn)
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## 3. Set the connection URL and run

With PostgreSQL running and the selected Bricks dependency already installed:

```sh
export MIGRATION_DSN='postgres://user:password@localhost:5432/disposable?sslmode=disable'
go run .
```

Replace the credentials and database name. URL-escape credentials, use deployment-appropriate TLS outside local development, and never print a secret-bearing URL. A first successful run creates the table. A second successful run leaves the schema unchanged and returns nil.

The migration package has no YAML configuration type. This command reads `MIGRATION_DSN` directly. If the application already stores connection settings in YAML, build an escaped URL from those settings as shown in the [PostgreSQL startup recipe](../../recipes/postgresql-migrations.md). Do not add an invented `app.migration` configuration block.

For an Fx variant with an application-owned filesystem group and explicit startup execution, follow the inline [PostgreSQL startup recipe](../../recipes/postgresql-migrations.md). For multiple module contributions and filename conventions, read the [shared guide](usage.md).

## Connection ownership and timeout limits

Bricks passes the URL to golang-migrate's registered PostgreSQL driver, not the Bricks database constructor. An existing GORM or `*sql.DB` connection is neither required nor reused. After successful migrate-instance construction, `Up` defers `m.Close()` to close the source and database driver. Close errors are discarded, so nil is not proof that close succeeded.

`Up` has no context argument or application-level timeout option. An Fx hook's context does not reach it. Configure supported connection parameters in the URL. The pinned driver accepts `x-statement-timeout` in milliseconds, defaulting to zero without a statement deadline. It uses PostgreSQL advisory locking; a blocked database lock is not cancelled by the Fx hook context.

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
