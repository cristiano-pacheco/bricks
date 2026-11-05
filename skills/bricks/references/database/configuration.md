# Configure a PostgreSQL connection

Read this guide for pool fields, TLS, direct constructors, DSNs, or detailed failures. For a first Fx connection, use [PostgreSQL setup](usage.md). Settings below belong to `github.com/cristiano-pacheco/bricks/pkg/database`.

## 1. Choose configuration fields

The module loads `app.database`. Omitted fields retain Go zero values before constructor behavior applies. The complete scalar configuration is also available in [base.yaml](../../examples/postgresql/config/base.yaml):

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: bricks_skill
    user: bricks_skill
    password: env://POSTGRES_PASSWORD
    sslmode: false
    sslcert: ""
    sslkey: ""
    sslrootcert: ""
    maxopenconnections: 10
    maxidleconnections: 2
    connecttimeout: 5
    statementtimeout: 5000
    locktimeout: 0
    idleintransaction: 0
    timezone: UTC
    applicationname: bricks_skill_connection
    searchpath: public
    preparestmt: false
    skipdefaulttransaction: false
    disableforeignkeyconstraint: false
    enablelogs: false
    prefersimpleprotol: false
```

| Fields | Behavior |
| --- | --- |
| `host`, `name`, `user` | Required nonempty strings. Validation does not trim them. |
| `port` | Required unsigned integer between 1 and 65535. |
| `password` | Optional to Bricks. An unset environment reference becomes empty. Authentication policy may require it. |
| `sslmode` | Boolean. False selects PostgreSQL `disable`; true selects `require`. Other SSL modes have no string setting here. |
| `sslcert`, `sslkey`, `sslrootcert` | Optional paths. The constructor appends them only when TLS is enabled. |
| `maxopenconnections` | Positive sets the pool limit. Zero or negative leaves the standard unlimited-open default. |
| `maxidleconnections` | Positive sets the idle limit. Zero or negative leaves the standard default of two, constrained by a positive open limit. Use `SetMaxIdleConns(0)` afterward to request no idle connections. |
| `connecttimeout` | Positive driver connection timeout in seconds. Zero or negative omits it. This is not an overall constructor deadline. |
| `statementtimeout`, `locktimeout`, `idleintransaction` | Positive PostgreSQL settings in milliseconds. Zero or negative leaves server policy in effect. |
| `timezone` | `DSN()` changes blank or whitespace-only values to `UTC`. |
| `applicationname`, `searchpath` | Optional strings. Empty values are omitted. |
| `preparestmt` | Enables GORM's prepared statement cache. Defaults to false. |
| `skipdefaulttransaction` | Skips GORM's default write transaction. Defaults to false. |
| `disableforeignkeyconstraint` | Changes GORM schema migration behavior, not existing constraints or SQL migration files. Defaults to false. |
| `enablelogs` | True assigns the Go `Logger` field to GORM. False does not install a silent logger. A nil logger leaves GORM's default behavior. |
| `prefersimpleprotol` | Exported spelling. The constructor does not use it. Setting it does not enable simple protocol. |

Bricks sets maximum connection lifetime to one hour and maximum idle age to ten minutes. There are no `ConnMaxLifetime`, `ConnMaxIdleTime`, `MaxRetries`, or `RetryDelay` fields or constructor options. Obtain the underlying pool with `db.DB()` to override its ages.

`Logger` and `NamingStrategy` are Go interfaces, not YAML settings. Loading YAML does not call another package's default constructor.

## 2. Set deployment TLS

Replace the corresponding base fields or load a TLS overlay:

```yaml
app:
  database:
    sslmode: true
    sslcert: /run/secrets/postgres-client.crt
    sslkey: /run/secrets/postgres-client.key
    sslrootcert: /run/secrets/postgres-ca.crt
```

The [TLS asset](../../examples/postgresql/config/tls.yaml) can be selected with `APP_ENV=tls` beside `base.yaml`. These fields select `require`; choose certificate and verification policy for the deployment rather than treating this as a complete security policy.

## 3. Supply Go-only settings when needed

Replace `database.Module` with these providers inside the application's `fx.New` options. This is a fragment. Import the Bricks database package, Fx, `gorm.io/gorm/logger`, and `gorm.io/gorm/schema`. Supply `password` from application secrets.

```go
fx.Provide(
    func() database.Config {
        return database.Config{
            Host: "localhost", Port: 5432, Name: "bricks_skill",
            User: "bricks_skill", Password: password,
            ConnectTimeout: 5, MaxOpenConnections: 10, MaxIdleConnections: 2,
            EnableLogs: true,
            Logger: logger.Default.LogMode(logger.Info),
            NamingStrategy: schema.NamingStrategy{TablePrefix: "app_"},
        }
    },
    database.NewWithLifecycle,
)
```

`NewWithLifecycle(cfg, lifecycle)` connects during construction and registers closure in `OnStop`. The logger must implement GORM's logger interface. The Bricks logger needs an adapter. There is no arbitrary `*gorm.Config` parameter. Construct GORM outside this package if these settings cannot express the required policy.

When repositories use `gorm.ErrDuplicatedKey`, set `db.TranslateError = true` before publishing the handle, or preserve the project's driver-code classification. The Bricks database configuration has no `TranslateError` field.

## 4. Construct without Fx when requested

This fragment belongs in a function returning `error`. Import `context`, `errors`, and the Bricks database package. The caller owns closure.

```go
cfg := database.Config{
    Host: "localhost", Port: 5432, Name: "bricks_skill", User: "bricks_skill",
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

Use a bounded application context for real queries. `sqlDB.Stats()` returns standard pool statistics. Exported `ConnectionStats` is only a data structure. Bricks has no accessor that populates it.

## DSNs and configuration helpers

`Config.Validate()` checks host, name, user, then port. `DSN()` builds the keyword DSN used by `New` without validation. `PostgresDSN()` validates and builds a `postgres://` URL. It includes credentials, database, SSL mode, time zone, and optional application name. It omits certificate paths, search path, and timeouts. Only an exactly empty time zone becomes UTC in that method.

Neither builder escapes interpolated values. Never log the result. For reserved characters, construct an escaped URL with `net/url` as in the [migration command](../../recipes/migration-command.md). `PostgresDSN()` does not preserve all connection policy.

`Clone()` copies all fields. Interface-valued fields still refer to the same implementations. `WithDatabase(name)` returns a copy with a new name without validating it.

## Errors and lifecycle limits

Use `errors.Is`. Invalid configuration wraps `ErrInvalidConfig` and the specific `ErrMissingHost`, `ErrMissingName`, `ErrMissingUser`, `ErrMissingPort`, or `ErrInvalidPortNumber`. Exhausted connection attempts and pool-access failures wrap `ErrConnectionFailed`.

The constructor makes at most three `gorm.Open` attempts with one-second and two-second waits. A background ten-second context limits retry waits, but GORM does not receive it. One open can exceed that time. Retry counts and delays are not configurable. Fx hook timeouts do not cancel constructor attempts.

Exported `ConnectionError` has `Host`, `Port`, `DB`, `Attempt`, and `Err` fields. Its `Error` formats them, while `Unwrap` and `Is` delegate to `Err`. These constructors do not instantiate that type, so ordinary failures need not contain it.

Normal lifecycle shutdown owns closure. An eagerly opened connection can still need cleanup if graph construction fails before lifecycle startup. Do not assume shutdown hooks cover every failed graph.
