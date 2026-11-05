# Use Chi's separate metrics listener

When your application already needs Chi, expose global collectors through Chi's built-in metrics listener instead of adding the independent listener module. The application HTTP listener and metrics listener have separate addresses. Chi uses `promhttp.Handler()`, so this choice gathers the default registry, not an injected custom gatherer.

## Start both listeners

1. In your existing Bricks application module, create `cmd/chi-metrics/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
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
		chi.Module,
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

   `metrics.Module` records one order into global collectors. `chi.Module` owns both listeners. There is deliberately no independent metrics listener module or custom gatherer provider here.

2. Create `cmd/chi-metrics/config/base.yaml`:

```yaml
app:
  http:
    address: 127.0.0.1:8080
    enablemetrics: true
    metricsaddress: 127.0.0.1:9091
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
```

   The addresses include ports. These are fields under `app.http`, not `app.metrics`. Keep the ports distinct. This minimal scenario configures only the listeners and their deadlines; full Chi configuration belongs to the server guide.

3. Run from the application root:

```sh
APP_CONFIG_DIR=cmd/chi-metrics/config APP_ENV=local go run ./cmd/chi-metrics
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:9091/metrics
```

   Run curl in another terminal. Expect `ok` from the health endpoint and `usecase_success_total{name="create_order"} 1` from the metrics endpoint. `/metrics` is not mounted on port 8080.

4. Press Ctrl-C. Chi drains both listeners. The application preserves shutdown errors. Create the Go file from the inline setup above and copy the [YAML](../examples/metrics-chi/config/base.yaml) if preferred.

## Alternatives and limits

Set `app.http.enablemetrics: false` to disable Chi's metrics listener while retaining its application listener. Do not install both this listener and the independent listener on the same address. A supplied registry elsewhere in the Fx graph does not change Chi's default handler. To expose custom or combined gatherers, disable Chi metrics and use the [independent listener](../references/http/server/metrics/usage.md) with explicit [registry wiring](metrics-registries.md).

For direct construction, create a Chi `Config` with `chi.Default()`, adjust its addresses, and call `chi.New`. The caller must own `Start()` and `Shutdown(ctx)`; direct `Start` blocks while serving, unlike Fx startup. This is an alternative server lifecycle, not another way to inject a gatherer. Route registration, CORS, Swagger and complete HTTP configuration are separate server tasks.
