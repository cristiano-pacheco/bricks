# Connect an application to PostgreSQL

Use `github.com/cristiano-pacheco/bricks/pkg/database` to provide a PostgreSQL `*gorm.DB`. This guide starts with YAML and Fx. The database and user must already exist. This module opens connections, not databases or schema migrations.

## 1. Create `config/base.yaml`

Start with this minimal local-development configuration, also supplied as [minimal.yaml](../../examples/postgresql/config/minimal.yaml). Save it as `config/base.yaml` and set `POSTGRES_PASSWORD` before starting the application.

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: bricks_skill
    user: bricks_skill
    password: env://POSTGRES_PASSWORD
```

This uses plaintext local connections and the default pool policy. For explicit timeouts, pool limits, and all other settings, use the full configuration below. `Logger` and `NamingStrategy` are Go interfaces, not YAML settings. Tracing and pool metrics are optional integrations, not part of this setup.

`host`, `name`, `user`, and a port between 1 and 65535 are required. The constructor does not require a password, but your PostgreSQL authentication policy may. An unset `POSTGRES_PASSWORD` becomes an empty string rather than a configuration-load error. Keep secrets out of logs.

The loader requires `base.yaml`. It uses the `config` directory unless `APP_CONFIG_DIR` selects another directory. For overlays, environment substitution, or loading only a subtree, read the [configuration guide](../config/usage.md).

## 2. Register the module and a database consumer

This complete `main.go` registers a startup query and lets Fx close the pool at shutdown:

```go
package main

import (
    "context"
    "fmt"

    "github.com/cristiano-pacheco/bricks/pkg/database"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

func registerQuery(lifecycle fx.Lifecycle, db *gorm.DB) {
    lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            var value int
            if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
                return fmt.Errorf("query PostgreSQL: %w", err)
            }
            return nil
        },
    })
}

func main() {
    fx.New(database.Module, fx.Invoke(registerQuery)).Run()
}
```

`database.Module` reads `app.database` and provides one unnamed `*gorm.DB`. The invoke requests that value, so Fx constructs it. A module with no consumer can remain unused because Fx providers are lazy.

The constructor connects before `OnStart`. Run application queries in startup hooks or later, not in constructors that assume migrations have finished. Do not install another unnamed database provider or another provider for `config.Config[database.Config]` alongside this module.

## 3. Start the application

From the application directory, with the selected Bricks dependency already installed:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
go run .
```

Successful startup means the database opened and `SELECT 1` succeeded. This version remains running until shutdown. Fx closes the underlying `*sql.DB` on normal shutdown. Keep one pool for the application rather than closing it after each request.

Create the application Go file from step 2. The retained [PostgreSQL assets](../../examples/postgresql/README.md) contain YAML configuration only. For schema migrations before queries or request handling, use the [startup recipe](../../recipes/postgresql-migrations.md).

## 4. Change TLS settings for your deployment

Replace the TLS fields in the local configuration when your deployment requires TLS. This is an alternative to `sslmode: false`, not another database connection:

```yaml
app:
  database:
    sslmode: true
    sslcert: /run/secrets/postgres-client.crt
    sslkey: /run/secrets/postgres-client.key
    sslrootcert: /run/secrets/postgres-ca.crt
```

This overlay is supplied as [tls.yaml](../../examples/postgresql/config/tls.yaml). Keep `base.yaml` and run with `APP_ENV=tls`, or replace the corresponding fields in your base file. `sslmode: true` selects PostgreSQL `require`; false selects `disable`. It does not expose a string setting for other SSL modes. Certificate paths are appended only when TLS is enabled. Choose certificate and verification policy for your deployment rather than treating this example as a complete security policy.

## Configuration reference

The distributed [base.yaml](../../examples/postgresql/config/base.yaml) shows every scalar field. Copy it instead of the minimal file when you want explicit connection policy:

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
    connecttimeout: 5 # Seconds per driver connection attempt.
    statementtimeout: 5000 # Milliseconds.
    locktimeout: 0 # Milliseconds. Zero leaves server policy in effect.
    idleintransaction: 0 # Milliseconds. Zero leaves server policy in effect.
    timezone: UTC
    applicationname: bricks_skill_connection
    searchpath: public
    preparestmt: false
    skipdefaulttransaction: false
    disableforeignkeyconstraint: false
    enablelogs: false
    prefersimpleprotol: false # Exported spelling. Currently has no effect.
```

Omitted fields retain Go zero values. The following constructor behavior applies after loading:

| YAML fields | Requirement, default, or behavior |
| --- | --- |
| `host`, `name`, `user` | Required nonempty strings. Validation does not trim them. |
| `port` | Required unsigned integer from 1 to 65535. |
| `password` | Optional to Bricks. Defaults to an empty string. |
| `sslmode` | Defaults to false. True selects `require`; false selects `disable`. |
| `sslcert`, `sslkey`, `sslrootcert` | Optional paths. Empty values are omitted. Used only with TLS enabled. |
| `maxopenconnections` | Positive values set the open limit. Zero or negative leaves the `database/sql` default of unlimited open connections. |
| `maxidleconnections` | Positive values set the idle limit. Zero or negative leaves the standard default of two idle connections, constrained by a positive open limit. To request zero idle connections, call `SetMaxIdleConns(0)` in Go afterward. |
| `preparestmt` | Defaults to false. Enables GORM's prepared-statement cache. |
| `skipdefaulttransaction` | Defaults to false. Skips GORM's default write transaction. |
| `disableforeignkeyconstraint` | Defaults to false. Changes foreign-key creation during GORM migration, not existing constraints or SQL migration files. |
| `enablelogs` | Defaults to false. True assigns the Go `Logger` field to GORM. False does not install a silent logger. A nil logger leaves GORM's default behavior. |
| `timezone` | Blank or whitespace-only becomes `UTC` in `DSN()`. |
| `applicationname`, `searchpath` | Optional strings, omitted when empty. |
| `statementtimeout`, `locktimeout`, `idleintransaction` | Positive integers become PostgreSQL settings in milliseconds. Zero or negative omits them and leaves server policy in effect. |
| `connecttimeout` | Positive integer passed to the driver in seconds. Zero or negative omits it. It is not an overall constructor deadline. |
| `prefersimpleprotol` | Defaults to false. Retains the exported typo and is unused by the constructor. Setting it does not enable simple protocol. |

Bricks always sets maximum connection lifetime to one hour and maximum idle age to ten minutes. There are no `ConnMaxLifetime`, `ConnMaxIdleTime`, `MaxRetries`, or `RetryDelay` configuration fields or constructor options. Obtain the underlying pool through `db.DB()` to override its ages.

## Alternative: supply configuration in Go

Use this when the application needs a GORM logger or naming strategy. Replace `database.Module` with your providers. This fragment belongs in an application's `fx.New` options:

```go
// Imports: database, go.uber.org/fx, gorm.io/gorm/logger,
// and gorm.io/gorm/schema. password is supplied by the application.
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

`NewWithLifecycle(cfg, lifecycle)` opens immediately and registers pool closure in `OnStop`. Its logger must implement `gorm.io/gorm/logger.Interface`. The Bricks logger is a different interface and needs an adapter. There is no arbitrary `*gorm.Config` parameter. Construct GORM outside this package if its settings cannot express your requirements.

## Alternative: construct without Fx

This fragment belongs in a function returning `error`. The caller owns pool closure:

```go
// Imports: context, errors, and the Bricks database package.
// password is supplied by the application.
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

Use a bounded application context for real operations. `sqlDB.Stats()` returns standard-library pool statistics. Exported `ConnectionStats` is only a data structure containing maximum, open, in-use and idle counts, wait count and duration, and idle/lifetime closure counts. Bricks has no accessor that populates it.

## Configuration helpers and DSNs

`Config.Validate()` checks `Host`, `Name`, `User`, then `Port`. `Config.DSN()` builds the keyword DSN used by `New` but does not validate it. `Config.PostgresDSN()` validates and builds a `postgres://` URL with host/port joining. The latter includes only credentials, database name, SSL mode, time zone, and optional application name. It omits certificate paths, search path, and timeouts. Only an exactly empty time zone becomes UTC in `PostgresDSN()`.

Neither builder escapes interpolated values. Spaces and reserved characters can change parsing or make the result invalid. Never log these strings. Use `net/url` for an escaped migration URL when needed, as the startup example does. Converting with `PostgresDSN()` does not preserve every connection policy.

`Clone()` copies all fields. Interface-valued `Logger` and `NamingStrategy` still refer to the same implementations. `WithDatabase(name)` returns that copy with a new `Name`, without validating it.

## Errors, retries, and lifecycle limits

Use `errors.Is` for constructor failures. Invalid configuration wraps both `ErrInvalidConfig` and the specific `ErrMissingHost`, `ErrMissingName`, `ErrMissingUser`, `ErrMissingPort`, or `ErrInvalidPortNumber`. Connection exhaustion and pool-access failures wrap `ErrConnectionFailed`.

The constructor makes at most three `gorm.Open` attempts, with one-second and two-second waits. A background ten-second context limits retry waits, but GORM does not receive it. An individual open can exceed it. Retry counts and delays are not configurable. Fx hook timeouts do not cancel constructor attempts.

The exported `ConnectionError` has `Host`, `Port`, `DB`, `Attempt`, and `Err` fields. Its `Error` formats that context; `Unwrap` and `Is` delegate to `Err`. These constructors do not instantiate it, so ordinary failures need not contain a `*ConnectionError`.

Normal lifecycle shutdown owns closure. An eagerly opened connection can still need cleanup if graph construction fails before lifecycle startup. Do not assume shutdown hooks cover every failed build.

## Optional compositions

The connection setup above needs no metrics, tracing, or SQLite guide. Ticket 14 owns these planned destinations, which are not yet distributed guides:

- Pool monitoring will use `recipes/database-pool-metrics.md` for `PoolMetricsModule`, registerer wiring, and collector lifecycle.
- GORM instrumentation will use `recipes/gorm-tracing.md` for tracing initialization and shutdown.
- Multiple connections will use `recipes/multiple-databases.md` for named providers. PostgreSQL and SQLite stock modules both provide unnamed `*gorm.DB` and collide.
