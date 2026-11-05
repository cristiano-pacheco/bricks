# Run PostgreSQL migrations before application startup

Use this recipe when queries, workers, or request handling must wait for SQL migrations. `database.Module` opens the application pool but does not apply SQL. The application registers migration execution and orders it before schema-dependent work.

## 1. Create the connection configuration

Create `config/base.yaml`:

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: bricks_skill
    user: bricks_skill
    password: env://POSTGRES_PASSWORD
    sslmode: false
    connecttimeout: 5 # Seconds per connection attempt.
    statementtimeout: 5000 # Milliseconds.
    maxopenconnections: 10
    maxidleconnections: 2
    applicationname: bricks_skill_startup
```

The database and user must already exist. The user needs permissions for application operations. The migration connection also needs DDL and migration-state permissions. TLS is disabled here only for local development. For every connection field, default, TLS alternative, and Go-only option, read the [database configuration reference](../references/database/configuration.md).

This recipe uses a separate `MIGRATION_DSN` environment variable for the migration URL. Point it at the same database and schema. It can use a distinct migration user. The migration package has no YAML configuration provider of its own.

## 2. Add the schema files

Create `migrations/000001_startup_items.up.sql` beside `main.go`:

```sql
CREATE TABLE bricks_skill_startup_items (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
```

Create `migrations/000001_startup_items.down.sql`:

```sql
DROP TABLE bricks_skill_startup_items;
```

Use a fresh database for this example. In an existing application, select an unused numeric version. Preserve historical files after deployment.

## 3. Register migrations before the application hook

This complete `main.go` checks the migrated table and then shuts down. Replace the table-count hook with your application startup only after preserving its `*schemaReady` dependency.

```go
package main

import (
    "context"
    "embed"
    "errors"
    "fmt"
    "io/fs"
    "os"
    "time"

    "github.com/cristiano-pacheco/bricks/pkg/database"
    "github.com/cristiano-pacheco/bricks/pkg/migration"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

//go:embed migrations/*.sql
var embedded embed.FS

type schemaReady struct{}

type migrationInputs struct {
    fx.In
    Files []migration.FileSystem `group:"application_migrations"`
}

func migrationFiles() (migration.FileSystem, error) {
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return migration.FileSystem{}, err
    }
    return migration.New(root), nil
}

func provideRunner(in migrationInputs) *migration.Runner {
    return migration.NewRunner(in.Files)
}

func registerMigrations(lifecycle fx.Lifecycle, runner *migration.Runner, _ *gorm.DB) (*schemaReady, error) {
    dsn := os.Getenv("MIGRATION_DSN")
    if dsn == "" {
        return nil, errors.New("MIGRATION_DSN is required")
    }
    lifecycle.Append(fx.Hook{
        OnStart: func(context.Context) error {
            if err := runner.Up(dsn); err != nil {
                return fmt.Errorf("apply PostgreSQL migrations: %w", err)
            }
            return nil
        },
    })
    return &schemaReady{}, nil
}

func registerApplication(lifecycle fx.Lifecycle, db *gorm.DB, _ *schemaReady) {
    lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            var count int64
            return db.WithContext(ctx).Table("bricks_skill_startup_items").Count(&count).Error
        },
    })
}

func run() error {
    app := fx.New(
        database.Module,
        fx.Provide(
            fx.Annotate(migrationFiles, fx.ResultTags(`group:"application_migrations"`)),
            provideRunner,
            registerMigrations,
        ),
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
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

`registerMigrations` consumes `*gorm.DB`, which forces the eager connection constructor and its close-hook registration first. It then registers the migration hook and returns `*schemaReady`. `registerApplication` requests that token before registering its own hook. Fx starts hooks in registration order and stops them in reverse order.

The token orders hook registration; it does not mean SQL has already run during construction. Query tables only in the later `OnStart` or afterward. A migration failure prevents that later hook from starting.

## 4. Run against a disposable database

With the application dependencies already selected and PostgreSQL running:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
export MIGRATION_DSN='postgres://bricks_skill:local-example-password@localhost:5432/bricks_skill?sslmode=disable&connect_timeout=5&statement_timeout=5000'
go run .
```

Successful startup applies the table migration before querying it. The program then stops and exits zero. A second successful run has no pending migrations and queries the existing table. The pool closes at shutdown. An Fx start timeout does not cancel migration execution, as explained below.

## Alternative: derive the migration URL from YAML

The retained [startup assets](../examples/postgresql-startup/README.md) contain configuration and SQL, not an alternate Go program. The inline setup above uses a separate `MIGRATION_DSN`; keep it aligned with the application connection settings.

## Add more module contributions

Each contributor returns `migration.New(rootFS)` through `fx.Annotate` with the same group tag. `fs.Sub` makes embedded SQL appear at the contribution root. The group name is application-owned; Bricks has no automatic migration group. Fx group order is unspecified. Give all contributions unique filenames and versions rather than relying on precedence.

For contribution layouts, filenames, collision rules, and public operation selection, read the [shared migration contract](../references/migration/usage.md). For PostgreSQL version state, dirty migrations, transaction boundaries, error wrapping, and absent rollback operations, read the [PostgreSQL migration branch](../references/migration/postgresql.md). This recipe owns ordering and connection composition, not those package contracts.

## Two connection owners

`database.Module` owns the application GORM pool and closes it on normal shutdown. `runner.Up(url)` opens a separate migration connection and defers its own close after successful migrate-instance construction. It does not reuse GORM or the application pool. Do not close GORM from the migration hook. Application pool limits do not configure the migration connection.

Use a migration URL built with `net/url` and explicit deployment TLS policy. If deriving the migration URL from typed database configuration in your application, preserve host, credentials, database name, SSL mode, certificates, time zone, search path, application name, and positive timeout settings. Pool and GORM fields have no URL equivalent. This guide does not distribute an alternate URL-building program. `Config.DSN()` is the application's keyword DSN, not this recipe's migration URL. `Config.PostgresDSN()` omits certificate and timeout settings and does not escape values; it is not a lossless conversion. For those constructor contracts, read [database configuration](../references/database/configuration.md) only when those contracts matter.

A deployment can instead supply a separately escaped migration URL with distinct privileges. Keep its target database and schema consistent with the application connection. Never print secrets or either DSN. Database permissions must permit the application's operations, while migration credentials must permit the required DDL and migration-state updates.

## Failure and shutdown

Configuration or database construction failures prevent startup. A migration error returns from its `OnStart` with context, causes `app.Start` to fail, and prevents later application hooks from running. The example calls `app.Stop` after any start attempt, with a fresh context, and joins errors rather than exiting before cleanup. Fx rolls back previously started hooks on startup failure. Successfully applied earlier PostgreSQL migrations may remain when a later migration fails. Startup failure is not a schema rollback.

The migration API accepts no context. An Fx start timeout does not cancel `runner.Up`, advisory locking, or SQL execution. Set suitable PostgreSQL connection and statement policies; use an externally controlled migration job when application startup needs a cancellable deployment boundary. The example supplies an Fx context deadline but does not present it as a hard migration deadline. Close errors from the runner's deferred close are discarded by its implementation.

If a serving hook starts successfully, its `OnStop` must stop accepting work before the database's close hook runs. The explicit readiness dependency keeps that application hook after migration registration. No serving hook is needed for the example's one-shot readiness query.

## Optional integrations

For pool monitoring, use [database pool metrics](database-pool-metrics.md). For GORM instrumentation, use [GORM tracing](gorm-tracing.md). For simultaneous database connections, use [multiple databases](multiple-databases.md). Basic migrations need none of these integrations.
