# Choose Chi's logging outputs

Enable built-in request and route logs, and send server route messages to a JSON slog handler. Bricks' server logger and Bricks' use-case logger are different dependencies.

## 1. Copy the server program and YAML

Copy [main.go](../../../../examples/chi-logging/main.go) to `cmd/logging-api/main.go` in your application module. Create `cmd/logging-api/config/base.yaml`:

```yaml
app:
  http:
    address: 127.0.0.1:8080
    enablemetrics: false
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
    enablerequestlogging: true
    enableroutelogging: true
```

The [distributed YAML](../../../../examples/chi-logging/config/base.yaml) matches this file. These flags are false when omitted from loaded YAML, unlike `chi.Default()`, which sets both true.

## 2. Understand the supplied logger

The copied program imports `log/slog`, `os`, Bricks server `chi`, and `go.uber.org/fx`. It installs this provider alongside its route contribution:

```go
func provideServerLogger() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stderr, nil))
}

func application() *fx.App {
    return fx.New(
        chi.Module,
        fx.Provide(provideServerLogger),
        fx.Provide(fx.Annotate(newGreetingRoutes, fx.ResultTags(`group:"routes"`))),
    )
}
```

This provider returns the exact optional `*slog.Logger` type consumed by `NewWithLifecycle`. Its stderr handler owns no file that needs closing. Route registration and the error-preserving Fx runner are already in the copied file.

The built-in request logger is go-chi's `middleware.Logger`. It does not use this supplied slog instance. A JSON server logger therefore does not turn those text request logs into JSON.

## 3. Start the application and issue a request

```sh
APP_CONFIG_DIR=cmd/logging-api/config APP_ENV=local go run ./cmd/logging-api
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/hello
```

Expect HTTP 200 with `{"data":{"message":"hello"}}`. Startup emits route messages through the JSON slog handler, and the request emits go-chi's text request log. Fx also has its own event logs. Press Ctrl-C to drain the listener and finish cleanup before the runner reports an error.

## Supported alternatives

Omit the slog provider to use `slog.Default()`. `chi.New` also uses `slog.Default()` for direct construction and has no logger argument or exported logger setter.

`logger.Module` supplies Bricks' `logger.Logger`, backed by Zap. It does not satisfy `*slog.Logger`. Conversely, a `*slog.Logger` is not Bricks' `logger.Logger`, which the response module and decorator factory require. Provide each exact contract needed by the application. The [complete HTTP flow](../../../../recipes/http-flow.md) supplies the Bricks logger separately without pretending it configures Chi logs.

For JSON access logs, set `enablerequestlogging: false` and install your own HTTP logging middleware on an application group. Use [the middleware guide](middleware.md) for safe insertion points. Application-group middleware does not wrap the built-in health route.

If your slog handler writes to an application-owned file, open it in a provider that returns errors and add an Fx stop hook to close it. Keep it alive until the server has stopped. There is no YAML key that binds a handler or converts Zap into slog.
