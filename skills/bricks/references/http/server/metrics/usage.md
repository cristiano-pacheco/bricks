# Serve an application-owned metrics registry

Expose `orders_created_total` at `http://127.0.0.1:9090/metrics` without an application HTTP server. Use an existing Go module with Bricks and its supported Go toolchain. This package has no module README.

## Start the independent listener

1. Create `cmd/metrics/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

func newGatherer() (prometheus.Gatherer, error) {
	registry := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total", Help: "Orders created by this application.",
	})
	if err := registry.Register(counter); err != nil {
		return nil, err
	}
	counter.Inc()
	return registry, nil
}

func run() error {
	app := fx.New(listener.Module, fx.Provide(newGatherer))
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

   `newGatherer` registers one application counter with its own registry and returns the exact interface the listener needs. This does not install the use-case metrics package. The listener's module forces construction and attaches start and stop hooks. Its optional logger dependency is `*slog.Logger`, not the Bricks logger interface. Without it, the listener uses `slog.Default()`.

2. Create `cmd/metrics/config/base.yaml`:

```yaml
app:
  metrics:
    enabled: true
    address: 127.0.0.1
    port: 9090
```

3. Run from your application's root:

```sh
APP_CONFIG_DIR=cmd/metrics/config APP_ENV=local go run ./cmd/metrics
```

   In another terminal:

```sh
curl -fsS http://127.0.0.1:9090/metrics
```

   Expect HTTP 200 and `orders_created_total 1`. This registry does not include `go_goroutines`, process collectors, or Bricks use-case collectors. Register them explicitly if you need them.

4. Press Ctrl-C to stop. `app.Stop` shuts down the HTTP server before main reports errors. The [distributed main.go](../../../../examples/metrics-listener/main.go) and [configuration files](../../../../examples/metrics-listener/config/base.yaml) are ready to copy.

## Complete configuration

The module loads `app.metrics` through the Bricks loader. It reads `base.yaml`, then optional `<APP_ENV>.yaml` from `APP_CONFIG_DIR`, which defaults to `config`. `APP_ENV` defaults to `local`. Duration strings have explicit units.

```yaml
app:
  metrics:
    enabled: true
    address: 127.0.0.1
    port: 9090
    readtimeout: 5s
    readheadertimeout: 10s
    writetimeout: 10s
    idletimeout: 60s
```

The matching [full.yaml](../../../../examples/metrics-listener/config/full.yaml) is a reference file. To use it, copy it to `base.yaml` in a separate configuration directory, or select it as the environment overlay with `APP_ENV=full`. The loader does not read every YAML file in the directory.

| Field | Type | Required, default, and constraints |
| --- | --- | --- |
| `enabled` | bool | Optional, false. False means no listener is bound. |
| `address` | string | Optional, empty binds all local interfaces. Host or IP only, not a `host:port` URL. |
| `port` | uint | Optional, zero asks the OS for an available port. Maximum 65535. |
| `readtimeout` | duration | Optional, zero means no overall request-read timeout. |
| `readheadertimeout` | duration | Optional. Fx replaces zero with 10s. Direct construction leaves zero unchanged, so net/http falls back to ReadTimeout. |
| `writetimeout` | duration | Optional, zero means no response-write timeout. |
| `idletimeout` | duration | Optional, zero uses net/http's ReadTimeout fallback. |

Use nonnegative durations. Bricks validates the port but does not validate timeout values. Restrict network access yourself; the listener does not add authentication or TLS.

For disabled serving, use this separate file, also [distributed](../../../../examples/metrics-listener/config/disabled.yaml):

```yaml
app:
  metrics:
    enabled: false
```

With `APP_ENV=disabled`, startup succeeds without binding and `Addr()` stays empty. A gatherer is still required during construction. An occupied enabled port fails startup synchronously. The application still waits for Ctrl-C in the example, even when serving is disabled.

## Direct ownership and custom handlers

This fragment replaces Fx-managed construction inside an application function. Add `context`, `errors`, `time`, and the aliased listener import from the complete example. The caller owns the returned server.

```go
server, err := listener.New(listener.Config{
    Enabled: true, Address: "127.0.0.1", Port: 9090,
    ReadHeaderTimeout: 10 * time.Second,
}, registry) // registry implements prometheus.Gatherer
if err != nil { return err }
if err = server.StartContext(ctx); err != nil {
    cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return errors.Join(err, server.Shutdown(cleanupCtx))
}
// Run application work, then call Shutdown with a fresh bounded context.
```

`NewWithHandler(cfg, handler)` accepts an `http.Handler` instead of a gatherer. For example, replace the `listener.New` call above with this fragment, adding the `github.com/prometheus/client_golang/prometheus/promhttp` import:

```go
server, err := listener.NewWithHandler(listener.Config{
    Enabled: true, Address: "127.0.0.1", Port: 9090,
    ReadHeaderTimeout: 10 * time.Second,
}, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

Keep the same error handling and start/stop ownership. The server still mounts that handler only at `/metrics`. A nil handler or nil gatherer fails construction. Registries, gatherers, handler options, authentication wrappers, and the optional logger are Go-only settings. There are no YAML keys for them.

`Start()` delegates to `StartContext(context.Background())`. `StartContext` binds before returning and honors a canceled startup context. Start once per instance. `Addr()` returns the bound address, useful for port zero. `Errors()` reports unexpected serving errors and closes when serving ends. Do not wait on it when serving is disabled, because it is not closed in that scenario.

The Fx module monitors `Errors()`, logs unexpected failures, and requests application shutdown with exit code 1. A direct owner must monitor the channel itself and handle shutdown. `Shutdown(ctx)` drains requests, attempts forced closure if draining fails, and preserves cleanup errors. A stopped instance cannot be restarted. `NewWithLifecycle` is available when you provide `config.Config[Config]`, an Fx lifecycle and shutdowner, and the gatherer yourself.

For global collectors and mixed registries, use [registry wiring](../../../../recipes/metrics-registries.md). For Chi's distinct listener choice, use [Chi metrics](../../../../recipes/metrics-chi.md).
