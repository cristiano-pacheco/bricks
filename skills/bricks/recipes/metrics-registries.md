# Connect collectors to the intended metrics listener

Choose which collectors a scrape should expose before adding a listener. A module does not merge Prometheus registries. These arrangements require no YAML beyond the listener configuration.

## Expose Bricks use-case collectors

1. In your existing application Go module with Bricks, create `cmd/registry-metrics/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

func registerOrder(lifecycle fx.Lifecycle, measurements metrics.UseCaseMetrics) {
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		started := time.Now()
		err := createOrder(ctx)
		measurements.ObserveDuration("create_order", time.Since(started))
		if err != nil {
			measurements.IncError("create_order")
			return err
		}
		measurements.IncSuccess("create_order")
		return nil
	}})
}

func createOrder(ctx context.Context) error {
	return ctx.Err()
}

func run() error {
	app := fx.New(
		metrics.Module,
		listener.Module,
		fx.Provide(func() prometheus.Gatherer { return prometheus.DefaultGatherer }),
		fx.Invoke(registerOrder),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	err := app.Start(startCtx)
	cancelStart()
	if err == nil {
		signal := <-app.Wait()
		if signal.ExitCode != 0 {
			err = fmt.Errorf("application requested exit code %d", signal.ExitCode)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

   The use-case constructor registers globally. The listener receives the corresponding default gatherer explicitly. The startup hook runs one operation and records its outcome.

2. Create `cmd/registry-metrics/config/base.yaml`:

```yaml
app:
  metrics:
    enabled: true
    address: 127.0.0.1
    port: 9090
```

3. Run from the application root, then scrape in another terminal:

```sh
APP_CONFIG_DIR=cmd/registry-metrics/config APP_ENV=local go run ./cmd/registry-metrics
curl -fsS http://127.0.0.1:9090/metrics
```

   Expect `usecase_success_total{name="create_order"} 1`. Ctrl-C stops the listener. It does not unregister collectors. Copy the [YAML](../examples/metrics-usecase/config/base.yaml) if preferred; create the Go file from step 1. A newly created registry will not contain these global collectors.

## Expose an application-owned registry

1. Starting with `main.go` above, remove `metrics.Module` and `fx.Invoke(registerOrder)` from `fx.New`. Remove `registerOrder`, `createOrder`, and the now-unused metrics import. Add this provider:

```go
func ownedGatherer() (prometheus.Gatherer, error) {
    registry := prometheus.NewRegistry()
    counter := prometheus.NewCounter(prometheus.CounterOpts{
        Name: "orders_created_total", Help: "Orders created by this application.",
    })
    if err := registry.Register(counter); err != nil { return nil, err }
    counter.Inc()
    return registry, nil
}
```

2. Replace the default gatherer provider with `fx.Provide(ownedGatherer)`. Retain `listener.Module` and the same lifecycle code.
3. Run with the same YAML and scrape command. Expect `orders_created_total 1` and no global use-case metrics. Register and gather from the same registry. Stop with Ctrl-C. Constructing a fresh registry does not change the global use-case constructor's registration behavior.

## Expose global and application collectors together

1. Start with the complete inline application under 'Expose Bricks use-case collectors' above. Add the following function to `main.go`. It uses that file's existing `prometheus` import.

```go
func combinedGatherer() (prometheus.Gatherer, error) {
    registry := prometheus.NewRegistry()
    counter := prometheus.NewCounter(prometheus.CounterOpts{
        Name: "orders_created_total", Help: "Orders created by this application.",
    })
    if err := registry.Register(counter); err != nil { return nil, err }
    counter.Inc()
    return prometheus.Gatherers{prometheus.DefaultGatherer, registry}, nil
}
```

2. Replace the example's gatherer provider with `fx.Provide(combinedGatherer)`. Leave `metrics.Module`, `listener.Module`, and the operation invoke unchanged. This combines gatherers without mutating process-wide default variables.
3. Run with the same configuration and scrape command. Expect both `orders_created_total 1` and `usecase_success_total{name="create_order"} 1`. Stop with Ctrl-C. Avoid duplicate family names across the gatherers, which can cause collection errors. Installing the use-case module twice still causes registration errors.

## Use a custom registerer for all collectors

Bricks' `NewPrometheusUseCaseMetrics` has no registerer argument. For application-owned use-case metrics, implement `metrics.UseCaseMetrics` using your registry and provide that interface instead of `metrics.Module`. Match the operations and seconds units documented in the [package guide](../references/metrics/usage.md). This is an application implementation, not an undocumented constructor option. Changing Prometheus default globals is process-wide and unsafe during concurrent registration; the recommended arrangements above avoid that change.

Use [Chi's built-in listener](metrics-chi.md) only when you want its default-gatherer endpoint. For pool collectors, select the database or Redis pool recipe when that task is requested. Do not load pool setup for a use-case-only scrape.
