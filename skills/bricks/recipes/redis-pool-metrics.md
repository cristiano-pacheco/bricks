# Expose Redis pool metrics

Use a Bricks Redis wrapper, adapt its underlying client for a pool collector, and expose those samples through an independent listener. You need an application module with Bricks, a local Redis server on port 6379, and port 9092 available.

## 1. Create the files

Create a directory inside your existing application Go module. Create `main.go` from the inline code below and copy [config/base.yaml](../examples/storage-redis-pool-metrics/config/base.yaml) into that directory. Keep the filenames shown here. Your application must already select a Bricks dependency; these files do not select or download another framework version.

## 2. Configure the components

Start your controlled local Redis deployment on localhost:6379 before running the example. Change the URL if your deployment uses another address. Create `config/base.yaml`:

```yaml
app:
  redis:
    type: single_node
    url: redis://localhost:6379
    metrics:
      enabled: true
  metrics:
    enabled: true
    address: 127.0.0.1
    port: 9092
```

`type` is a string selecting `single_node` here. An omitted type selects that same default. `url` is the required Redis URL. This local example has no authentication or TLS and uses database zero. Server authentication, addresses, and topology must match your deployment.

`app.redis.metrics.enabled` is a boolean pool-collector switch, false by default. It differs from `app.redis.enablemetrics`, which enables the wrapper's local counters and does not register a Prometheus collector. Neither switch changes the listener configuration.

`app.metrics.enabled` is a boolean, false when omitted. `address` is a host string, not `host:port`. `port` is an unsigned integer from 0 to 65535, with zero selecting a dynamic port. The example selects loopback port 9092 so the scrape command has a stable target. Omitted read, write, and idle timeouts are zero. The Fx listener sets an omitted read-header timeout to 10 seconds. Registry objects and Fx bindings are Go-only settings.

## 3. Add the Go wiring

`ClientModule` owns one Bricks wrapper. `underlying` exposes that same connection as `redis.UniversalClient`, the type required by `PoolMetricsModule()`. The registry is supplied through both Prometheus interfaces. Do not also install `redis.Module`, which would construct another client. The operation calls `PING` through the adapted client; no application data is written. Create `main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
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

func underlying(client *bricksredis.Client) bricksredis.UniversalClient {
	return client.UniversalClient()
}

func registerWork(lc fx.Lifecycle, client bricksredis.UniversalClient) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		value, err := client.Ping(ctx).Result()
		if err != nil {
			return fmt.Errorf("ping Redis: %w", err)
		}
		fmt.Println("redis=" + value)
		return nil
	}})
}

func run() error {
	app := fx.New(
		bricksredis.ClientModule,
		fx.Provide(underlying, newRegistry),
		bricksredis.PoolMetricsModule(),
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
APP_ENV=local APP_CONFIG_DIR=./config go run .
```

The application prints `redis=PONG` and stays running. In another terminal:

```sh
curl --fail http://127.0.0.1:9092/metrics
```

Expect `redis_pool_conn_total_current`, `redis_pool_conn_idle_current`, and `redis_pool_hit_total` samples. Counts depend on the client pool, not the total Redis server connections. Press Ctrl-C to stop the listener, unregister the collector, and close the wrapper.

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

## Universal clients, topologies, and cleanup

If the application needs no wrapper helpers, replace `ClientModule` with `Module` and remove `underlying` from `fx.Provide`. Keep the remaining graph unchanged. The universal interface is a type alias of go-redis's `UniversalClient`, so no second conversion or second client is needed.

`PoolMetricsModule()` reads enablement while Fx options are built. It requires a nonnil universal client and registerer when enabled. It registers a `redisprometheus.NewCollector("redis", "", client)` at startup and unregisters it at shutdown. The client owner still closes the connection. Duplicate pool collectors with the same metric descriptions in one registry reject registration. Choose a distinct namespace or subsystem through the direct `redisprometheus.NewCollector` API when writing your own collector lifecycle for multiple clients.

The collector reads `PoolStats` through the universal interface. It does not impose the tracing module's `*redis.Client` restriction. Cluster pool statistics come from go-redis's aggregate client statistics, not labeled per-node series. Sentinel and failover statistics describe the selected data client, not Sentinel server health. Start with exactly one selected topology's configuration; do not merge standalone, cluster, and Sentinel options into one file.

## Troubleshooting and optional configuration

For deployment-specific configuration, load only the selected [standalone](../references/redis/standalone.md), [cluster](../references/redis/cluster.md), [Sentinel](../references/redis/sentinel.md), or [failover](../references/redis/failover.md) guide. For namespaced keys or local counters, load the [wrapper guide](../references/redis/client.md). For listener alternatives, load the [independent listener guide](../references/http/server/metrics/usage.md).

Run from the application directory where you created `main.go` and `config/`, or set `APP_CONFIG_DIR` to an absolute path. The loader merges `base.yaml` with the selected environment file. Set `APP_ENV=local` as shown to avoid accidentally selecting a deployment overlay. These examples have not been compiled or executed as part of this documentation change; the results above are expected observations, not recorded test results.
