# Start a Chi application server

Serve `GET /hello` through Bricks' Chi server and let Fx own startup and shutdown. Use an application Go module that already depends on Bricks. No database, metrics registry, logger module, or configuration provider is needed for this setup.

## 1. Create the server configuration

Create `cmd/hello/config/base.yaml`:

```yaml
app:
  http:
    address: 127.0.0.1:8080
    enablemetrics: false
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
```

`address` includes the port. Metrics are explicitly disabled for this application. The module reads `app.http` and does not merge `chi.Default()` into the YAML. These timeout values are explicit settings, not loader defaults. Omitted logging flags are false.

## 2. Add a route and the application

Create `cmd/hello/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"go.uber.org/fx"
)

type GreetingRoutes struct{}

func newGreetingRoutes() chi.Route { return &GreetingRoutes{} }

func (r *GreetingRoutes) Setup(server *chi.Server) {
	server.Router().Get("/hello", func(w http.ResponseWriter, request *http.Request) {
		if err := response.JSON(w, http.StatusOK, map[string]string{"message": "hello"}, nil); err != nil {
			slog.ErrorContext(request.Context(), "write greeting", "error", err)
		}
	})
}

func application() *fx.App {
	return fx.New(
		chi.Module,
		fx.Provide(fx.Annotate(newGreetingRoutes, fx.ResultTags(`group:"routes"`))),
	)
}

func run(app *fx.App) error {
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
	if err := run(application()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`newGreetingRoutes` returns the `chi.Route` interface and contributes it to the Fx `routes` group. A provider returning only `*GreetingRoutes` must instead add `fx.As(new(chi.Route))` to its annotation. `chi.Module` collects that group, calls each route's `Setup` during startup, and binds the listener before startup succeeds.

The callback logs response-write errors because it cannot return an error to `net/http`. It does not try to write a second response after a write failure. Application startup and shutdown errors reach `main` only after cleanup.

## 3. Start the application and send requests

Run from your application root:

```sh
APP_CONFIG_DIR=cmd/hello/config APP_ENV=local go run ./cmd/hello
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/hello
curl -i http://127.0.0.1:8080/healthz
```

Expect HTTP 200 with `{"data":{"message":"hello"}}` from `/hello`, and HTTP 200 with `ok` from `/healthz`. `/healthz` is built in. `/metrics` is not a public application route.

## 4. Stop the application

Press Ctrl-C. The runner calls `app.Stop` with a fresh context. The Chi lifecycle hook drains enabled listeners. A positive `shutdowntimeout` adds an internal deadline, while the caller's earlier deadline still wins. The process reports startup and cleanup errors through stderr and a nonzero exit code.

Create the Go file from step 2 and copy the [base YAML](../../../../examples/chi-basic/config/base.yaml) if preferred. The setup needs only this skill and your existing Bricks dependency.

## Supported alternatives

Without Fx, call `chi.New(cfg)`, register handlers on `server.Router()`, call `SetupRoutes()` once, and run the blocking `Start()`. Another goroutine must own `Shutdown(ctx)`. Own signal handling in your application and preserve both serving and cleanup errors. With an ephemeral bind, wait for startup before reading `Addr()`. Direct construction has code-only configuration and no YAML requirement. Treat `http.ErrServerClosed` from normal shutdown as success.

`RegisterRoute(route)` and `RegisterRoutes(routes)` append to the server's registry. `SetupRoutes()` delegates to `RouteRegistry.SetupAll(server)` and installs Swagger when enabled. Direct callers must call it before `Start`; Fx does that automatically. `NewRouteRegistry`, `Add`, and `SetupAll` are available when an application owns a separate registry. The registry does not deduplicate entries. Set routes up once, before serving.

`Router()` returns `*github.com/go-chi/chi/v5.Mux`. You can register handlers directly instead of implementing `Route`. `Addr()` reports the configured address before startup and the bound address afterward. `MetricsAddr()` does the same for enabled metrics and returns an empty string when disabled. Port `0` selects an available port. A server instance cannot restart after shutdown.

When assembling your own lifecycle module, `NewWithLifecycle` takes `NewWithLifecycleParams` with `Config config.Config[chi.Config]`, `LC fx.Lifecycle`, `Shutdowner fx.Shutdowner`, `Routes []chi.Route` in group `routes`, and an optional `Logger *slog.Logger`. The supplied configuration wrapper exposes its value through `Get()`. The stock `chi.Module` already provides the configuration, constructor, and forcing invoke. Adding another unnamed provider for the same type creates a collision.

## When startup fails

`New` validates configuration and wraps invalid settings with `ErrInvalidConfig`. Address and port failures expose `ErrInvalidAddress`, `ErrInvalidMetricsAddress`, `ErrInvalidPort`, or `ErrInvalidMetricsPort`. Equal nonzero listener port numbers expose `ErrPortsEqual`, even if the hosts differ. A failed bind exposes `ErrListen`. Starting twice exposes `ErrServerStarted`. Use `errors.Is` to inspect wrapped errors.

Fx startup binds both enabled listeners synchronously. If the metrics bind fails, Chi closes the public listener before returning. Later metrics serving errors are logged and request application shutdown with exit code 1. Unexpected main-listener errors after Fx startup are currently discarded by the asynchronous serving method. Direct `Start` returns main-listener serving errors to its caller, but only logs later metrics serving errors. Do not describe either mode as complete process-health supervision.

## Load optional setup only when needed

- For every supported YAML field, direct defaults, and functional options, use [configuration](configuration.md).
- For browser origins or credentials, use [CORS](cors.md).
- For additional route middleware or proxy IP policy, use [middleware](middleware.md).
- For request logs, route logs, or a supplied slog logger, use [logging](logging.md).
- For an API documentation UI and its required document registration, use [Swagger](swagger.md).
- For the separate default-registry metrics listener, use [Chi metrics](../../../../recipes/metrics-chi.md). For custom gatherers, use the independent listener recipe linked there.
- For body decoding, use-case validation, explicit decoration, and error responses together, use [the HTTP flow](../../../../recipes/http-flow.md).
- For list-query policy and response metadata, use [pagination and responses](../../../../recipes/pagination-response.md).
