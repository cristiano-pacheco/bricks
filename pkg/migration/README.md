# Migration Package

A database migration package that provides co-located, per-module SQL migration support using embedded filesystems and `golang-migrate`, with Uber FX integration.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

### Standalone (no FX)

Collect `FileSystem` values from each module's migrations sub-package and pass them to `NewRunner`:

```go
package main

import (
    "log"

    "github.com/cristiano-pacheco/bricks/pkg/migration"
    catalogmigrations "your-app/internal/modules/catalog/migrations"
    profilemigrations "your-app/internal/modules/profile/migrations"
)

func main() {
    runner := migration.NewRunner([]migration.FileSystem{
        migration.New(catalogmigrations.FS),
        migration.New(profilemigrations.FS),
    })

    if err := runner.Up("postgres://user:pass@localhost:5432/mydb?sslmode=disable"); err != nil {
        log.Fatal(err)
    }
}
```

### SQLite

`Runner.UpSQLite` accepts the underlying `*sql.DB` from the Bricks SQLite
package. It applies every pending up migration and its version update in one
transaction. The supplied database remains open and owned by the caller.

```go
package main

import (
    "log"

    "github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
    "github.com/cristiano-pacheco/bricks/pkg/migration"
)

func main() {
    db, err := sqlite.New(sqlite.Config{DSN: "./data/workplan.db"})
    if err != nil {
        log.Fatal(err)
    }
    sqlDB, err := db.DB()
    if err != nil {
        log.Fatal(err)
    }
    defer sqlDB.Close()

    runner := migration.NewRunner(nil)
    if err := runner.UpSQLite(sqlDB); err != nil {
        log.Fatal(err)
    }
}
```

`DownSQLite` is an explicit development-only operation that applies matching
down migrations in reverse order. Runtime startup should call only
`UpSQLite`; it never downgrades automatically. SQLite migrations must not
contain explicit `BEGIN` or `COMMIT` statements because the runner owns the
transaction. A database version newer than the migrations in the running
binary is rejected with `ErrDatabaseNewer`.

### With Uber FX

Each module provides a `migration.FileSystem` tagged to the `"migration_filesystems"` group. A consumer (e.g. a CLI command or startup hook) collects them all and runs the migrations:

```go
// In each module's fx.go:
fx.Annotate(
    func() migration.FileSystem {
        return migration.New(catalogmigrations.FS)
    },
    fx.ResultTags(`group:"migration_filesystems"`),
),
```

## Features

- **Co-located migrations**: SQL files live inside each module's own directory, co-located with the code they serve
- **Embedded SQL**: Uses `//go:embed` — no filesystem path dependency at runtime
- **Global ordering**: Migration files from all modules are merged into a single sorted virtual FS; lexicographic timestamp prefixes guarantee correct cross-module execution order
- **ErrNoChange handling**: `Runner.Up` returns `nil` when there are no pending migrations
- **Transactional SQLite**: `Runner.UpSQLite` applies all pending up migrations and records the final version in one transaction
- **Explicit development downgrades**: `Runner.DownSQLite` applies matching down migrations only when called directly
- **FX Integration**: First-class support for the Uber FX `group:"migration_filesystems"` value group pattern

## SQL File Naming

Migration filenames must follow `golang-migrate`'s convention:

```
<version>_<description>.up.sql
<version>_<description>.down.sql
```

The `<version>` prefix is typically a UTC timestamp. Because all modules share a single global version sequence, timestamps must be unique across modules:

```
20260216115959_create_categories_table.up.sql    ← catalog
20260216120001_create_profiles_table.up.sql      ← profile
20260217120000_create_ai_prompt_templates.up.sql ← ai
```

## Module Layout

Each module that owns database tables contains a dedicated migrations sub-package:

```
internal/modules/<module>/
  migrations/
    migrations.go          ← embeds *.sql
    <ts>_<desc>.up.sql
    <ts>_<desc>.down.sql
```

**`migrations/migrations.go`:**

```go
package catalogmigrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

The sub-package exists to keep `cmd/db_migrate.go` free of heavy module transitive dependencies (GORM, Chi, etc.). Only the embed is imported.

## API

### `FileSystem`

A thin wrapper around `fs.FS` used to carry a module's embedded migrations:

```go
type FileSystem struct {
    fs.FS
}

func New(f fs.FS) FileSystem
```

### `Runner`

```go
type Runner struct { /* ... */ }

func NewRunner(filesystems []FileSystem) *Runner
```

Creates a runner from a slice of `FileSystem` values. Modules are merged in the order provided; because each file is uniquely named (timestamp prefix), ordering of filesystems has no effect on the global migration sequence.

### `Runner.Up`

```go
func (r *Runner) Up(dsn string) error
```

Merges all filesystems, creates a `golang-migrate` instance via the `iofs` driver, and applies all pending up-migrations. Returns `nil` on success or when there are no new migrations (`ErrNoChange`).

### `Runner.UpSQLite` and `Runner.DownSQLite`

```go
func (r *Runner) UpSQLite(db *sql.DB) error
func (r *Runner) DownSQLite(db *sql.DB) error
```

`UpSQLite` reads and validates all contributed migration files before opening a
transaction, rejects a dirty or newer schema, applies pending up migrations in
ascending version order, and commits once. Errors roll back the entire run.
`DownSQLite` is for explicit development use and applies all matching down
migrations in descending version order in one transaction. Neither method
closes `db`.

## Complete FX Integration Example

### Step 1: Create the migrations sub-package

```
internal/modules/catalog/migrations/migrations.go
```

```go
package catalogmigrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

Add SQL files alongside it:

```
internal/modules/catalog/migrations/
  20260216115959_create_categories_table.up.sql
  20260216115959_create_categories_table.down.sql
  20260216120002_create_products_table.up.sql
  20260216120002_create_products_table.down.sql
```

### Step 2: Provide the FileSystem in the module's fx.go

```go
import (
    bricksmigration "github.com/cristiano-pacheco/bricks/pkg/migration"
    catalogmigrations "your-app/internal/modules/catalog/migrations"
    "go.uber.org/fx"
)

func provideCatalogMigrationFS() bricksmigration.FileSystem {
    return bricksmigration.New(catalogmigrations.FS)
}

var Module = fx.Module(
    "catalog",
    fx.Provide(
        fx.Annotate(
            provideCatalogMigrationFS,
            fx.ResultTags(`group:"migration_filesystems"`),
        ),
        // ... other providers
    ),
)
```

### Step 3: Run migrations in the CLI command

```go
// cmd/db_migrate.go
package cmd

import (
    "fmt"
    "log/slog"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/config"
    "github.com/cristiano-pacheco/bricks/pkg/database"
    bricksmigration "github.com/cristiano-pacheco/bricks/pkg/migration"
    aimigrations      "your-app/internal/modules/ai/migrations"
    catalogmigrations "your-app/internal/modules/catalog/migrations"
    profilemigrations "your-app/internal/modules/profile/migrations"
    "github.com/spf13/cobra"
)

var dbMigrateCmd = &cobra.Command{
    Use:   "db:migrate",
    Short: "Run database migrations",
    RunE: func(_ *cobra.Command, _ []string) error {
        logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

        cfg, err := config.New[database.Config](config.WithPath("app.database"))
        if err != nil {
            return fmt.Errorf("load config: %w", err)
        }
        dsn, err := cfg.Get().PostgresDSN()
        if err != nil {
            return fmt.Errorf("build dsn: %w", err)
        }

        runner := bricksmigration.NewRunner([]bricksmigration.FileSystem{
            bricksmigration.New(catalogmigrations.FS),
            bricksmigration.New(aimigrations.FS),
            bricksmigration.New(profilemigrations.FS),
        })

        if err = runner.Up(dsn); err != nil {
            return fmt.Errorf("run migrations: %w", err)
        }

        logger.Info("migrations completed")
        return nil
    },
}
```

## Dependencies

This package depends on:

- [`pkg/database`](../database) - For `Config.PostgresDSN()` used when building the DSN
- [`pkg/database/sqlite`](../database/sqlite) - For the caller-owned SQLite connection used by `UpSQLite`
- [`github.com/golang-migrate/migrate/v4`](https://github.com/golang-migrate/migrate) - Migration engine
- [`github.com/golang-migrate/migrate/v4/source/iofs`](https://github.com/golang-migrate/migrate/tree/master/source/iofs) - In-memory FS source driver
- `database/sql` (standard library) - caller-owned SQLite connection for transactional migrations
- `testing/fstest` (standard library) - `MapFS` used as the merged virtual filesystem
