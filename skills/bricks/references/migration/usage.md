# Add SQL migrations to an application

Use `github.com/cristiano-pacheco/bricks/pkg/migration` to collect SQL files owned by application modules and apply them in numeric version order. The application supplies the files and explicitly starts migration execution. Bricks has no migration Fx module or automatic startup migration.

## 1. Create an up/down pair

Put the files beside the Go package that embeds them:

```text
migrations/
  000001_create_accounts.up.sql
  000001_create_accounts.down.sql
main.go
```

For a small example, use this up file:

```sql
CREATE TABLE migration_example_accounts (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
```

Use this down file:

```sql
DROP TABLE migration_example_accounts;
```

Reserve a unique version across every application module. An up/down pair shares one version, but two up files must not share it. Keep historical files after deployment. The runner sends SQL to the selected backend without interpolation or dialect conversion.

## 2. Embed the directory and construct a runner

This application fragment includes all imports needed for file collection. Add the backend call from step 3 inside `run`:

```go
package main

import (
    "embed"
    "io/fs"

    "github.com/cristiano-pacheco/bricks/pkg/migration"
)

//go:embed migrations/*.sql
var embedded embed.FS

func run() error {
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return err
    }
    runner := migration.NewRunner([]migration.FileSystem{
        migration.New(root),
    })
    // Add one backend operation here. This is a setup fragment.
    _ = runner
    return nil
}
```

`fs.Sub` makes the SQL files appear at the filesystem root. The runner reads that root and skips directories. It never searches subdirectories. For multiple modules, append each wrapped filesystem to the slice, with unique versions across them.

## 3. Execute the operation for your backend

Choose one branch. The runner itself has no YAML configuration type, field defaults, or configuration provider.

For PostgreSQL, supply an escaped connection URL and call `Up`:

```go
// Fragment inside run. dsn comes from application configuration.
return runner.Up(dsn)
```

`Up` opens its own migration connection. It does not require or reuse a Bricks database constructor, GORM, or an existing pool. For a complete direct application, URL inputs, dirty state, and collision behavior, read [PostgreSQL execution](postgresql.md). The [migration assets](../../examples/migration-postgresql/README.md) retain SQL only.

For SQLite, pass an already opened `*sql.DB`:

```go
// Fragment inside run. The application owns db and closes it afterward.
return runner.UpSQLite(db)
```

`UpSQLite` leaves the handle open. For a complete application that owns its handle, transactions, version validation, and development rollback, read [SQLite execution](sqlite.md). The [migration assets](../../examples/migration-sqlite/README.md) retain SQL only.

Return migration failures to the caller. Apply migrations before starting requests or workers that use the schema. Serialize schema changes in the deployment workflow.

## Alternative: collect module contributions with Fx

Use this when application modules register their migration files independently. Add these declarations alongside the embedding code above:

```go
// Fragment. Additional import: go.uber.org/fx.
func migrationFiles() (migration.FileSystem, error) {
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return migration.FileSystem{}, err
    }
    return migration.New(root), nil
}

type migrationInputs struct {
    fx.In
    Files []migration.FileSystem `group:"application_migrations"`
}

func provideRunner(in migrationInputs) *migration.Runner {
    return migration.NewRunner(in.Files)
}

var migrationOptions = fx.Options(
    fx.Provide(
        fx.Annotate(migrationFiles,
            fx.ResultTags(`group:"application_migrations"`)),
        provideRunner,
    ),
)
```

`application_migrations` is an application-chosen group name, not a Bricks contract. Every contributor and consumer must use the same name. Fx group order is unspecified, so it cannot resolve conflicting files.

Providers collect files but do not execute migrations. Register an invoke or `OnStart` hook that consumes the runner and calls the chosen operation. For PostgreSQL migrations ordered before application readiness, read the [startup recipe](../../recipes/postgresql-migrations.md).

## Filename and filesystem reference

`New(f fs.FS) FileSystem` wraps the filesystem in its exported `FS` field. `NewRunner(filesystems []FileSystem) *Runner` stores the slice without validation or execution. Supply non-nil filesystems and keep the slice and file contents unchanged while using the runner. Each execution reads them again.

The filename parser accepts a decimal version, underscore, identifier, lowercase `up` or `down`, and an extension. It does not enforce `.sql` or a timestamp width. Use positive numeric versions and `.sql` files. Versions sort numerically, not by identifier or contribution order. Down SQL must undo up SQL in reverse dependency order.

Keep unrelated files outside the migration root. Filesystem read errors return through execution. PostgreSQL and SQLite handle malformed files and collisions differently, so consult only the backend branch you use when diagnosing those failures.

## Public operations and limits

| Operation | Input and ownership |
| --- | --- |
| `Runner.Up(dsn)` | PostgreSQL URL. Opens a separate migration connection and closes it after successful migrate-instance construction. |
| `Runner.UpSQLite(db)` | Caller-owned SQLite `*sql.DB`. Applies pending up files and leaves the handle open. |
| `Runner.DownSQLite(db)` | Explicit development rollback on a caller-owned SQLite handle. Never runs automatically. |

There is no public PostgreSQL rollback, generic `Down`, version query, force-version operation, or runner close method. A down file alone does not make PostgreSQL rollback available. `UpSQLite` does not call `DownSQLite` on failure or shutdown.
