# Connect an application to SQLite

Use `github.com/cristiano-pacheco/bricks/pkg/database/sqlite` for a SQLite `*gorm.DB` with foreign-key enforcement and one connection per pool. This setup opens a file, checks its foreign-key policy, and closes it through Fx. You need the selected Bricks dependency in your application module, CGO, and a C compiler. No database server is required.

## 1. Create `config/base.yaml`

```yaml
app:
  sqlite:
    path: ./bricks-skill.db
```

The file path is relative to the process working directory, not the configuration directory. SQLite creates the database file, but not missing parent directories. Create those directories first and grant the process write access. Use a disposable path for these examples.

`sqlite.Module` loads `app.sqlite` through the Bricks configuration loader. It requires `base.yaml` in `config`, unless `APP_CONFIG_DIR` selects another directory. `APP_ENV` selects an optional overlay and defaults to `local`. The minimal file is also distributed as [minimal.yaml](../../../examples/sqlite/config/minimal.yaml).

## 2. Add `main.go`

This complete application checks the real connection during startup, then stops. Fx owns connection cleanup.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

func registerQuery(lifecycle fx.Lifecycle, db *gorm.DB) {
    lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            var enabled int
            if err := db.WithContext(ctx).Raw("PRAGMA foreign_keys").Scan(&enabled).Error; err != nil {
                return fmt.Errorf("read foreign-key policy: %w", err)
            }
            if enabled != 1 {
                return errors.New("foreign-key enforcement is disabled")
            }
            fmt.Println("foreign_keys=1")
            return nil
        },
    })
}

func run() error {
    app := fx.New(sqlite.Module, fx.Invoke(registerQuery))
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

The invoke requests the lazy `*gorm.DB` provider. Construction opens and pings SQLite before `OnStart`; `NewWithLifecycle` registers the close hook. Application stop hooks should finish database work before that hook runs. Do not close the pool after each operation.

## 3. Run the application

```sh
APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run .
```

Expect `foreign_keys=1`, Fx lifecycle messages, and exit status zero. The database file remains, but this connection check creates no application tables. The [copyable example](../../../examples/sqlite/README.md) contains the same program and all YAML alternatives below.

For real schema SQL, an ordered migration hook, an application query, and explicit development rollback, follow the independently loadable [SQLite migration startup recipe](../../../recipes/sqlite-migrations.md). Opening this module alone never applies SQL migrations.

## Alternative connection scenarios

Each YAML block below replaces `config/base.yaml`. Do not use these as overlays on the path example, because overlays retain `path` and the constructor rejects both fields together.

For a driver DSN with explicit file creation, use [dsn.yaml](../../../examples/sqlite/config/dsn.yaml):

```yaml
app:
  sqlite:
    dsn: 'file:./bricks-skill.db?mode=rwc'
```

For a named in-memory database, use [memory.yaml](../../../examples/sqlite/config/memory.yaml):

```yaml
app:
  sqlite:
    dsn: 'file:bricks-skill-memory?mode=memory&cache=shared'
    journal_mode: MEMORY
```

`:memory:` is also a supported DSN, but each underlying connection gets a separate database. Named memory with `cache=shared` lets connections using the same name share it within a process. Memory databases disappear when the last connection closes. One open and one idle connection reduce accidental loss; closing the pool still destroys it. WAL is not available for an in-memory database, so select MEMORY rather than expecting the file-backed WAL result.

The constructor rewrites `_foreign_keys`, `_journal_mode`, `_busy_timeout`, and `_txlock` in the DSN. It removes their `_fk`, `_journal`, and `_timeout` aliases. A DSN cannot override those typed settings. Other driver query parameters remain available, such as `mode=ro` for an existing read-only database. Default WAL may require writes, so choose a compatible journal policy and permissions for a read-only deployment.

## Direct construction instead of Fx

Replace `run` in the complete application above with this function. Remove the unused `time`, Fx, and GORM imports. Keep `context`, `errors`, `fmt`, `os`, and `sqlite`.

```go
func run() (err error) {
    db, err := sqlite.New(sqlite.Config{Path: "./bricks-skill.db"})
    if err != nil {
        return err
    }
    sqlDB, err := db.DB()
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, sqlDB.Close()) }()
    return sqlDB.PingContext(context.Background())
}
```

`New` does not load YAML and transfers pool ownership to the caller. To load the same typed YAML yourself, replace its construction line with this fragment and add the `github.com/cristiano-pacheco/bricks/pkg/config` import:

```go
loaded, err := config.New[sqlite.Config](config.WithPath("app.sqlite"))
if err != nil {
    return err
}
db, err := sqlite.New(loaded.Get())
```

For custom Fx wiring, `NewWithLifecycle(lifecycle, config.Config[sqlite.Config])` returns the same database with an `OnStop` close hook. `sqlite.Module` already provides both that constructor and the typed configuration. Install custom providers instead of duplicating them.

## Configuration reference

This full scalar configuration is distributed as [base.yaml](../../../examples/sqlite/config/base.yaml):

```yaml
app:
  sqlite:
    path: ./bricks-skill.db
    foreign_keys: true
    journal_mode: WAL
    busy_timeout: 5s
    tx_lock: immediate
    max_open_connections: 1
    max_idle_connections: 1
    prepare_stmt: false
    skip_default_transaction: false
    disable_foreign_key_constraint_when_migrating: false
    enable_logs: false
```

Only the connection selector is required. All other YAML fields are optional. Values in the table describe `New`, which applies defaults before validation. `Config.Validate()` alone validates without applying defaults or opening a connection.

| YAML field | Type | Required status, default, and constraints |
| --- | --- | --- |
| `path` | string | Exactly one of `path` and `dsn` must contain a non-whitespace value. No default. A filesystem path is the simplest choice. |
| `dsn` | string | Alternative to `path`, with no default. Accepts a path or SQLite driver DSN, including memory URIs. Setting both fails rather than preferring one. |
| `foreign_keys` | bool | Optional. `New` forces true, even if false is supplied. Foreign keys cannot be disabled through this configuration or DSN. |
| `journal_mode` | string | Empty becomes WAL. Accepts DELETE, TRUNCATE, PERSIST, MEMORY, WAL, OFF, case-insensitively and with surrounding whitespace trimmed. Actual mode also depends on SQLite storage and permissions. OFF removes journaling protection. |
| `busy_timeout` | duration string | Zero becomes 5s. Positive values must be at least 1ms. Negative values fail. The driver receives whole milliseconds, truncating finer precision. This bounds lock waits, not the whole operation. |
| `tx_lock` | string | Empty becomes immediate. Accepts deferred, immediate, exclusive, case-insensitively with whitespace trimmed. Immediate transactions acquire the write reservation at begin and may wait or fail under contention. |
| `max_open_connections` | integer | Zero becomes 1. Only 0 and 1 pass validation, so unlimited or multi-connection pools are unavailable. |
| `max_idle_connections` | integer | Zero becomes 1. Only 0 and 1 pass validation. Zero does not disable idle connections. |
| `prepare_stmt` | bool | Defaults false. Enables GORM prepared-statement caching. |
| `skip_default_transaction` | bool | Defaults false. True skips GORM's default write transaction, not explicit transactions or migration transactions. |
| `disable_foreign_key_constraint_when_migrating` | bool | Defaults false. True skips foreign-key generation by GORM AutoMigrate. It does not disable SQLite enforcement or alter SQL migration files. |
| `enable_logs` | bool | Defaults false, selecting GORM's discard logger unless a custom logger exists. True leaves the GORM logger choice in place. It does not configure a Bricks application logger. |

### Go-only GORM settings

Configure `GORMConfig *gorm.Config`, `Logger logger.Interface`, and `NamingStrategy schema.Namer` in Go. Although `GORMConfig` has a `config:"gorm_config"` tag, it exposes a third-party struct with functions and interfaces, not a stable Bricks YAML schema. The full YAML covers the supported scalar settings; use Go for custom GORM behavior.

This optional fragment replaces the direct constructor's configuration argument:

```go
sqlite.Config{
    Path: "./bricks-skill.db",
    GORMConfig: &gorm.Config{TranslateError: true},
    NamingStrategy: schema.NamingStrategy{TablePrefix: "app_"},
}
```

Add `gorm.io/gorm` and `gorm.io/gorm/schema` for this fragment. The constructor shallow-copies `GORMConfig`. It ORs the convenience booleans with its corresponding GORM values, so false cannot turn off a true GORM setting. `PrepareSTMT` and `DisableForeignKeyConstraint` are Go-only compatibility aliases for the corresponding convenience fields. An explicit `Logger` or `NamingStrategy` overrides the copied GORM choice. A custom GORM logger remains active even when `enable_logs` is false.

`DefaultConfig(dsn)` fills the runtime defaults and sets `DSN`. Clear `DSN` before switching that result to `Path`. A zero `Config` has no connection selector and fails construction.

## Constraints and failure handling

Foreign-key enforcement only checks constraints present in the schema. Declare real references in SQL or let GORM generate them. Disabling GORM constraint creation does not make existing constraints inactive.

The pool has one connection, not one connection for the entire machine. Other processes or other pools can still contend. While holding a transaction, use its transaction handle for all related queries. Calling the outer pool from inside the transaction can wait indefinitely for the sole connection. The busy timeout addresses SQLite locks, not waiting for a free Go pool connection. Close result rows before starting another operation that needs that connection.

Use `errors.Is` for `ErrInvalidConfig` and its causes. Missing selectors produce `ErrMissingDSN`, dual selectors `ErrMultipleDSNs`, and malformed query escaping `ErrInvalidDSN`. Invalid settings produce `ErrInvalidBusyTimeout`, `ErrInvalidJournalMode`, `ErrInvalidTxLock`, `ErrInvalidMaxOpenConnections`, or `ErrInvalidMaxIdleConnections`. Open and ping failures wrap `ErrConnectionFailed` and preserve their underlying cause. Lifecycle close errors also propagate. Missing directories, permissions, locks, and CGO/toolchain failures need their own diagnosis rather than retrying every error.

## Optional integrations

For SQL migration contracts and rollback failure details, load the [SQLite migration branch](../../migration/sqlite.md). For loader overlays or environment references, load the [configuration guide](../../config/usage.md). Neither is needed for the basic connection check.

PostgreSQL and SQLite modules both provide an unnamed `*gorm.DB`; installing both creates a collision. Multiple named connections, pool metrics, and GORM tracing belong to the planned ticket 14 recipes. Those recipes are not distributed yet. Keep those integrations separate from basic connection setup and do not assume module installation supplies them.
