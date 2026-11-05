# Expose PostgreSQL pool metrics

Run a PostgreSQL query and expose its connection-pool statistics on an independent metrics listener. You need an application Go module with Bricks, an existing local PostgreSQL database, and loopback port 9092 available.

## 1. Create the files

Create a directory inside your existing application Go module. Create `main.go` from the inline code below and copy [config/base.yaml](../examples/storage-database-pool-metrics/config/base.yaml) into that directory. Keep the filenames shown here. Your application must already select a Bricks dependency; these files do not select or download another framework version.

## 2. Configure the components

Set `POSTGRES_PASSWORD` in your shell to the password of the existing `bricks_skill` PostgreSQL user. That user must have access to an existing database named `bricks_skill` on localhost:5432. Change the connection fields if your controlled development database uses different values. Create `config/base.yaml`:

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: bricks_skill
    user: bricks_skill
    password: env://POSTGRES_PASSWORD
    sslmode: false
    metrics:
      enabled: true
  metrics:
    enabled: true
    address: 127.0.0.1
    port: 9092
```

`host`, `name`, and `user` are required nonempty strings. `port` is a required integer from 1 to 65535. `password` is an optional string, empty by default. `env://POSTGRES_PASSWORD` reads that environment variable while loading configuration. `sslmode` is a boolean, false by default. These examples disable TLS only for a local development database. Names and credentials must match an existing database. PostgreSQL field names follow the exported struct names without separators, unlike SQLite's explicit underscore tags.

`app.database.metrics.enabled` is a boolean switch, false by default. It is read by `PoolMetricsModule()`, not by the connection constructor. It is separate from all GORM connection fields.

`app.metrics.enabled` is a boolean, false when omitted. `address` is a host string, not `host:port`. `port` is an unsigned integer from 0 to 65535, with zero selecting a dynamic port. The example selects loopback port 9092 so the scrape command has a stable target. Omitted read, write, and idle timeouts are zero. The Fx listener sets an omitted read-header timeout to 10 seconds. Registry objects and Fx bindings are Go-only settings.

## 3. Add the Go wiring

The same `prometheus.Registry` is provided as both `Registerer` and `Gatherer`. `database.PoolMetricsModule()` consumes the unnamed `*gorm.DB` and the registerer. The listener consumes the gatherer. Merely adding these modules does not connect unrelated registries. The query hook runs after the collector and listener startup hooks. Create `main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/database"
	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type registryValues struct {
	fx.Out
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
}

func newRegistry() registryValues {
	registry := prometheus.NewRegistry()
	return registryValues{Registerer: registry, Gatherer: registry}
}

func query(ctx context.Context, db *gorm.DB) error {
	var value int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
		return fmt.Errorf("query PostgreSQL: %w", err)
	}
	if value != 1 {
		return fmt.Errorf("unexpected query result: %d", value)
	}
	fmt.Println("postgres=1")
	return nil
}

func registerWork(lc fx.Lifecycle, db *gorm.DB) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return query(ctx, db)
	}})
}

func run() error {
	app := fx.New(
		database.Module,
		fx.Provide(newRegistry),
		database.PoolMetricsModule(),
		listener.Module,
		fx.Invoke(registerWork),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()
	if startErr == nil {
		signal := <-app.Wait()
		if signal.ExitCode != 0 {
			startErr = fmt.Errorf("application requested exit code %d", signal.ExitCode)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
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

The `run` function returns errors after Fx shutdown. `main` reports any failure and exits nonzero only after cleanup. The stop context is fresh, so a cancelled startup or operation context does not prevent cleanup.

## 4. Run and observe the result

Run from the directory containing `main.go`:

```sh
APP_ENV=local APP_CONFIG_DIR=./config POSTGRES_PASSWORD="$POSTGRES_PASSWORD" go run .
```

The application prints `postgres=1` and stays running. In another terminal, scrape the listener:

```sh
curl --fail http://127.0.0.1:9092/metrics
```

Expect `go_sql_open_connections{db_name="postgres"}` and related `go_sql_` samples. Values depend on the live pool. The fixed `db_name` label is `postgres`, not the configured database name. Press Ctrl-C to stop the listener, unregister the collector, and close the pool.

## Use the process default registry instead

Replace only `newRegistry` with this fragment. Keep the existing `registryValues` type and listener wiring:

```go
func newRegistry() registryValues {
    return registryValues{
        Registerer: prometheus.DefaultRegisterer,
        Gatherer: prometheus.DefaultGatherer,
    }
}
```

This registers the pool collector globally rather than in the isolated registry. Do not register the same collector twice. Global use-case metrics do not appear in an isolated registry automatically. When you need both isolated pool collectors and global use-case collectors, keep the isolated registerer and return `prometheus.Gatherers{registry, prometheus.DefaultGatherer}` as its gatherer. Do not combine a registry with itself. See [registry arrangements](metrics-registries.md) only when composing those sources.

## Ownership and additional databases

`PoolMetricsModule()` loads its enablement while building Fx options. Configuration must exist before `fx.New`. When enabled, it requires an initialized unnamed `*gorm.DB` and a nonnil `prometheus.Registerer`. It constructs a Prometheus `DBStatsCollector`, registers it during startup, and unregisters that collector during shutdown. A duplicate registration fails startup. The module does not own the database or registry. `database.Module` closes the database.

The module hardcodes the `postgres` collector label and does not select Fx names. For multiple named databases, write a consumer hook that injects the named handles and calls `collectors.NewDBStatsCollector(sqlDB, uniqueName)` for each. Register each collector in `OnStart`, return registration errors, and unregister it in `OnStop`. This direct collector API also works without Fx. It can observe SQLite, but do not install the PostgreSQL convenience module and present its hardcoded label as a SQLite identity.

## Troubleshooting and optional configuration

For full PostgreSQL YAML and TLS choices, load the [database guide](../references/database/usage.md). For listener timeout and disabled-serving scenarios, load the [independent metrics listener guide](../references/http/server/metrics/usage.md). For named clients, load [multiple databases](multiple-databases.md).

Run from the application directory where you created `main.go` and `config/`, or set `APP_CONFIG_DIR` to an absolute path. The loader merges `base.yaml` with the selected environment file. Set `APP_ENV=local` as shown to avoid accidentally selecting a deployment overlay. These examples have not been compiled or executed as part of this documentation change; the results above are expected observations, not recorded test results.
