# Measure a use case

Record a successful order and expose its Prometheus sample on a separate listener. Use an application Go module that already depends on Bricks and its supported Go toolchain. This package has no YAML configuration. The listener has its own configuration.

## Set up the recommended global-registry path

1. Create a directory such as `cmd/order-metrics` in your application. Add `main.go` below. `metrics.Module` binds `metrics.UseCaseMetrics`. The listener consumes an explicitly provided `prometheus.Gatherer`. Both use the Prometheus defaults in this example.

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

   `createOrder` represents application work. In a real use case, validate input at the beginning, then perform the operation. Record its duration once and increment exactly one outcome counter. The startup hook calls it once so the scrape has a predictable result. The main function preserves startup and shutdown errors.

2. Create `cmd/order-metrics/config/base.yaml`:

```yaml
app:
  metrics:
    enabled: true
    address: 127.0.0.1
    port: 9090
```

   The listener binds only loopback. Registries and collector names are Go dependencies, not YAML settings.

3. From the application's root, run:

```sh
APP_CONFIG_DIR=cmd/order-metrics/config APP_ENV=local go run ./cmd/order-metrics
```

   In another terminal, request the listener:

```sh
curl -fsS http://127.0.0.1:9090/metrics
```

   Expect `usecase_success_total{name="create_order"} 1` and `usecase_duration_seconds_count{name="create_order"} 1`. The duration sum varies. The error series does not exist until `IncError` runs. Go and process collectors also appear because the default gatherer includes them.

4. Press Ctrl-C in the application terminal. Fx stops the listener. Collectors remain registered for the lifetime of this process. The [distributed example](../../examples/metrics-usecase/main.go) and [configuration](../../examples/metrics-usecase/config/base.yaml) contain these files for copying. No framework checkout is needed.

## Direct construction

Replace `metrics.Module` with this provider in the same application. The interface binding is necessary because Fx does not infer it from the returned concrete pointer.

```go
// Fragment for the fx.New options in main.go.
fx.Provide(fx.Annotate(metrics.NewPrometheusUseCaseMetrics,
    fx.As(new(metrics.UseCaseMetrics)))),
```

Without Fx, call `metrics.NewPrometheusUseCaseMetrics()`, propagate its error, and call the same methods around your operation. It does not own a goroutine, expose an HTTP endpoint, or require shutdown. Own any HTTP listener separately.

## Methods and collector ownership

`NewPrometheusUseCaseMetrics` registers three vectors through `prometheus.Register`, which uses `prometheus.DefaultRegisterer`. It does not accept an application registerer. `Module` provides the constructor lazily; a consumer such as `registerOrder` must request the interface. Installing the module does not decorate use cases automatically.

| Method | Result |
| --- | --- |
| `ObserveDuration(name, duration)` | Adds `duration.Seconds()` to `usecase_duration_seconds`, a histogram. |
| `IncSuccess(name)` | Increments `usecase_success_total`. |
| `IncError(name)` | Increments `usecase_error_total`. |

Every vector has the label `name`. Use stable operation names, not user IDs or request IDs. Histogram bucket bounds in seconds are `0.005, 0.025, 0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 3, 4, 5`, plus Prometheus's infinite bucket.

Repeated construction normally returns an already-registered error. It does not reuse the first collector. Registration is sequential, so a later registration failure leaves earlier successful registrations behind. There is no unregister or reset method on the wrapper. Do not build one instance per request or assume stopping an Fx application unregisters it.

## Select another integration only when needed

For an application-owned registry or a combined gatherer, use [collector-to-listener wiring](../../recipes/metrics-registries.md). For listener fields, custom handlers, or disabled serving, load the [independent listener guide](../http/server/metrics/usage.md). For an application that already uses Chi and needs its built-in listener, use [Chi metrics](../../recipes/metrics-chi.md). None of these listeners makes separately registered collectors share a registry.
