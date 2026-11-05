# PostgreSQL connections

Use `github.com/cristiano-pacheco/bricks/pkg/database` for a PostgreSQL `*gorm.DB`. Connections open during construction, not in an Fx `OnStart` hook. Constructors validate configuration, open with retries, and configure the underlying `*sql.DB` pool.

## Construction and ownership

- `database.Module` loads `config.Config[database.Config]` from `app.database` and provides one unnamed, lifecycle-managed `*gorm.DB`. Include a consumer or invoke to force Fx's lazy provider to run. Do not also provide that configuration wrapper or an unnamed database yourself.
- `NewWithLifecycle(cfg, lifecycle)` accepts a typed `database.Config`, connects immediately, and appends an `OnStop` hook that obtains and closes the underlying `*sql.DB`. Use an application provider for programmatically supplied settings instead of installing `Module` too.
- `New(cfg)` returns the same configured connection without lifecycle registration. The caller owns closing its underlying pool.

A complete Fx application, YAML input, startup query, and shutdown are in the [connection example](../../examples/postgresql/README.md). Load it when a complete application is needed. There is no arbitrary `*gorm.Config` constructor parameter or options API. Customize the fields described below, or construct GORM outside this package for settings it cannot express.

This direct-construction fragment belongs inside a function returning `error`, with `context`, `errors`, and `database` imported:

```go
cfg := database.Config{
    Host: "localhost", Port: 5432, Name: "app", User: "app",
    Password: password, ConnectTimeout: 5,
    MaxOpenConnections: 10, MaxIdleConnections: 2,
}
db, err := database.New(cfg)
if err != nil {
    return err
}
sqlDB, err := db.DB()
if err != nil {
    return err
}
queryErr := sqlDB.PingContext(context.Background())
return errors.Join(queryErr, sqlDB.Close())
```

Use a bounded application context in real operations. Do not close the pool after each request. Lifecycle construction registers shutdown but does not make opening lazy until startup or make constructors context-aware. Fx hook timeouts do not cancel constructor connection attempts. Eagerly opened resources also require care on graph-construction failures before lifecycle startup; do not assume normal shutdown hooks cover every failed build.

## Required settings and errors

`Config.Validate()` checks `Host`, `Name`, `User`, and `Port` in that order. Empty strings fail, zero port fails, and a port above 65535 fails. It does not trim required strings or require a password. `New` wraps validation failures with both `ErrInvalidConfig` and the specific `ErrMissingHost`, `ErrMissingName`, `ErrMissingUser`, `ErrMissingPort`, or `ErrInvalidPortNumber`. Use `errors.Is`.

Connection exhaustion and pool-access failures wrap `ErrConnectionFailed`. There are at most three `gorm.Open` attempts with one-second and two-second waits between them. A background ten-second context limits retry waits, but GORM does not receive that context. An individual open can exceed it. Set positive `ConnectTimeout` for the driver's connection timeout in seconds; it is not an overall constructor deadline. Retry counts and delays are not configurable.

The exported `ConnectionError` has `Host`, `Port`, `DB`, `Attempt`, and `Err` fields. Its `Error` formats that context; `Unwrap` and `Is` delegate to `Err`. These constructors do not instantiate it. Do not require `errors.As` to find a `*ConnectionError` in ordinary constructor failures.

## Configuration and DSNs

The module requires a configuration directory with `base.yaml`, defaulting to `config`, or selected through `APP_CONFIG_DIR`. Supply lowercase field names beneath `app.database`, as shown in the complete example. For overlays, `env://` resolution, and whole-document versus subtree loading, read the [configuration guide](../config/usage.md) only when those behaviors need explanation. YAML supplies data, not interface implementations. Set `Logger` and `NamingStrategy` in Go.

`Config.DSN()` builds the keyword DSN used by `New`. It does not call `Validate`. `SSLMode` defaults to false and selects `disable`; true selects `require`, not a configurable SSL mode string. `SSLCert`, `SSLKey`, and `SSLRootCert` are appended only with SSL enabled. Set deployment-appropriate TLS settings explicitly. A blank or whitespace-only `TimeZone` becomes `UTC`.

`ApplicationName` and `SearchPath` are appended when nonempty. Positive `StatementTimeout`, `LockTimeout`, and `IdleInTransaction` become PostgreSQL settings in milliseconds. Positive `ConnectTimeout` is in seconds. Zero or negative values omit those parameters and leave server or driver policy in effect. The constructor does not validate them further.

`Config.PostgresDSN()` validates and builds a `postgres://` URL with host/port joining. It includes only credentials, database name, SSL mode, time zone, and optional application name. It omits certificate paths, search path, and timeouts. Unlike `DSN`, only an exactly empty time zone becomes UTC. Neither builder escapes interpolated values. Spaces and reserved characters can change parsing or make a DSN invalid. Avoid logging either form, and construct a properly escaped URL with `net/url` when migration credentials contain reserved characters. Do not assume converting to `PostgresDSN` preserves connection policy.

`Clone()` copies every field, but its interface-valued `Logger` and `NamingStrategy` remain shared implementations, not recursively copied objects. `WithDatabase(name)` returns that copy with a different `Name`; it does not validate the new name. `PreferSimpleProtol` retains its exported spelling but is unused when constructing the PostgreSQL driver. It does not enable simple protocol.

## Pool and GORM settings

Only positive `MaxOpenConnections` and `MaxIdleConnections` call `SetMaxOpenConns` and `SetMaxIdleConns`. Zero and negative values leave `database/sql` defaults, rather than setting the pool to zero idle connections. The standard defaults are unlimited open connections and two idle connections, with idle capacity constrained by any positive open limit. Set different policies afterward on the underlying `*sql.DB` when necessary.

Bricks always sets maximum connection lifetime to one hour and maximum idle age to ten minutes. There are no `ConnMaxLifetime`, `ConnMaxIdleTime`, `MaxRetries`, or `RetryDelay` configuration fields. Override pool ages explicitly through `db.DB()` if required. `sqlDB.Stats()` returns live standard-library statistics. Exported `ConnectionStats` is only a data structure, with maximum/open/in-use/idle counts, wait count and duration, and idle/lifetime closure counts; no Bricks accessor populates it.

`PrepareSTMT` enables GORM's prepared-statement cache. `SkipDefaultTransaction` skips GORM's default write transaction. `DisableForeignKeyConstraint` affects foreign-key creation during GORM migration, not existing database constraints or SQL migration files. All default to false. A non-nil `NamingStrategy` replaces GORM's naming strategy.

`EnableLogs` only assigns `Config.Logger` into GORM when true. False does not explicitly install a silent logger. A nil logger leaves GORM's default behavior. Supply a `gorm.io/gorm/logger.Interface` when customizing it; the Bricks logger interface is not that interface. Choose an adapter or GORM logger explicitly.

## Optional compositions

For SQL migrations before application readiness, read the [PostgreSQL startup recipe](../../recipes/postgresql-migrations.md). A connection alone does not execute migrations.

Ticket 14 owns these planned destinations. Load them only for the stated task after they exist:

- For pool monitoring, `recipes/database-pool-metrics.md` owns `PoolMetricsModule`, registerer wiring, and collector lifecycle.
- For GORM instrumentation, `recipes/gorm-tracing.md` owns tracing initialization and shutdown.
- For simultaneous database connections, `recipes/multiple-databases.md` owns named providers. PostgreSQL and SQLite stock modules both provide unnamed `*gorm.DB` and collide.

Basic connection setup needs none of those integrations. These are planned pointers, not existing guides or verified examples.
