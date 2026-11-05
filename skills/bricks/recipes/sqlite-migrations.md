# Run SQLite migrations before application queries

Use this recipe when an Fx application must apply SQL before querying SQLite. It opens one Bricks-managed pool, passes its underlying `*sql.DB` to the migration runner, queries a migrated table, and stops. You need an application module with the selected Bricks dependency, CGO, and a C compiler. No server or external migration connection is needed.

## 1. Create `config/base.yaml`

Use a fresh disposable file for this example:

```yaml
app:
  sqlite:
    path: ./sqlite-startup.db
```

The directory must exist. This path is relative to the process working directory. `sqlite.Module` loads `app.sqlite` from `base.yaml`, opens SQLite with foreign keys, WAL, a five-second lock timeout, immediate transactions, and one open and one idle connection. The module owns pool closure. The migration package has no YAML schema or separate DSN for this operation.

## 2. Create the SQL files

Create `migrations/000001_create_items.up.sql` beside `main.go`:

```sql
CREATE TABLE bricks_skill_items (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
INSERT INTO bricks_skill_items (id, name) VALUES (1, 'example');
```

Create `migrations/000001_create_items.down.sql`:

```sql
DROP TABLE bricks_skill_items;
```

Use an unused numeric version in an existing application. Preserve deployed files. Keep explicit transaction commands out of SQL files because the SQLite runner owns the transaction.

## 3. Add ordered startup in `main.go`

This complete program includes an explicit development-only rollback flag. Normal startup applies pending migrations; normal shutdown only closes connections.

```go
package main

import (
    "context"
    "embed"
    "errors"
    "flag"
    "fmt"
    "io/fs"
    "os"
    "time"

    "github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
    "github.com/cristiano-pacheco/bricks/pkg/migration"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

//go:embed migrations/*.sql
var embedded embed.FS

type schemaReady struct{}

func provideRunner() (*migration.Runner, error) {
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return nil, err
    }
    return migration.NewRunner([]migration.FileSystem{migration.New(root)}), nil
}

func run(down bool) error {
    registerMigrations := func(lifecycle fx.Lifecycle, runner *migration.Runner, db *gorm.DB) (*schemaReady, error) {
        sqlDB, err := db.DB()
        if err != nil {
            return nil, err
        }
        lifecycle.Append(fx.Hook{
            OnStart: func(context.Context) error {
                if down {
                    return runner.DownSQLite(sqlDB)
                }
                return runner.UpSQLite(sqlDB)
            },
        })
        return &schemaReady{}, nil
    }
    registerApplication := func(lifecycle fx.Lifecycle, db *gorm.DB, _ *schemaReady) {
        lifecycle.Append(fx.Hook{
            OnStart: func(ctx context.Context) error {
                if down {
                    fmt.Println("development rollback complete")
                    return nil
                }
                var count int64
                if err := db.WithContext(ctx).Table("bricks_skill_items").Count(&count).Error; err != nil {
                    return fmt.Errorf("query migrated table: %w", err)
                }
                fmt.Printf("items=%d\n", count)
                return nil
            },
        })
    }
    app := fx.New(
        sqlite.Module,
        fx.Provide(provideRunner, registerMigrations),
        fx.Invoke(registerApplication),
    )
    if err := app.Err(); err != nil {
        return err
    }
    startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
    startErr := app.Start(startCtx)
    cancelStart()
    stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelStop()
    return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
    down := flag.Bool("down", false, "undo all migrations on a disposable development database")
    flag.Parse()
    if err := run(*down); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

The migration provider consumes `*gorm.DB`, forcing database construction and close-hook registration before it registers its migration hook. The application requests `*schemaReady`, forcing migration-hook registration before its own query hook. Fx starts hooks in registration order. The token orders registration, not SQL execution during construction. Keep schema-dependent work in the later hook or afterward.

The runner receives the exact `*sql.DB` from GORM. It neither opens a separate connection pool nor closes this one. Do not close it in the migration hook. When adding serving components, register their hooks after the migration dependency and stop serving before the pool's close hook.

## 4. Apply, repeat, and explicitly roll back

Run from the directory containing `main.go`:

```sh
APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run .
APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run .
```

Expect `items=1` and exit status zero on both runs, alongside Fx messages. The first run creates `bricks_skill_items` and records version 1 in `schema_migrations`. The second does not insert another row. The pool closes and the file remains.

Only against disposable development data, run:

```sh
APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run . -down
```

Expect `development rollback complete`. `DownSQLite` undoes all available applied up versions in descending order, not just one step. It requires their matching down files. Here it drops `bricks_skill_items` and clears the current version while retaining the metadata table. Run normally to recreate the table and sample row. For a command without Fx, use the [direct migration example](../examples/migration-sqlite/README.md).

All files above are distributed together in the [SQLite startup example](../examples/sqlite-startup/README.md). It is a copying convenience; the numbered instructions contain the complete setup.

## Failure behavior and ownership

SQLite loads and validates contributions before entering the migration transaction. Pending up SQL and version changes share one transaction, so a failed pending migration rolls back that batch. Previously committed migrations remain. Down operations likewise use one transaction. A migration failure stops later application hooks; it is not a request to run down migrations.

After any start attempt, this application calls `app.Stop` with a fresh context and joins startup and shutdown errors. Fx also rolls back already-started hooks after a start failure. The SQLite module closes its pool; the runner leaves it open. Ordinary failures reach `main` only after cleanup.

Neither SQLite migration operation takes a context. The runner uses `context.Background()`, so the Fx startup deadline does not cancel migration SQL or waiting for a pool connection. The busy timeout only controls SQLite lock waits. Serialize migration calls and avoid holding a transaction or open rows while invoking the runner against the one-connection pool.

For naming, filesystem contributions, and shared public operations, read the [migration contract](../references/migration/usage.md) when adding contributors. For dirty/newer database rejection, metadata limitations, missing down files, and error sentinels, read the [SQLite migration branch](../references/migration/sqlite.md). These references own those contracts; this recipe owns startup composition.

## Alternatives and optional integrations

For direct construction, use `sqlite.New`, obtain `db.DB()`, and defer closing that handle while joining close errors. Pass it to `UpSQLite` before application queries. The [SQLite guide](../references/database/sqlite/usage.md) gives that ownership pattern, full-field YAML, memory and DSN alternatives, and Go-only GORM settings.

For several migration contributors, return `migration.FileSystem` values into an application-owned Fx group, collect the group in a runner provider, and keep the same readiness dependency. Group order does not define migration order. The shared migration contract gives the complete group pattern.

Multiple database connections, pool metrics, and GORM tracing remain separate integrations planned for ticket 14. Their recipes are not yet distributed. In particular, adding both database modules without naming their `*gorm.DB` results causes an Fx collision. Do not add instrumentation or a second database just to run these migrations.
