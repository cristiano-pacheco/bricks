# Add route middleware

Attach application middleware to a Chi route group without changing built-in routes. This example adds `X-Example: bricks` to `/hello`, but not to `/healthz`.

## 1. Create the application

Create `cmd/middleware-api/main.go` from the complete inline application in [server usage](usage.md). In step 2 below, replace its `GreetingRoutes.Setup` method and add the header middleware. Add the `chiv5 "github.com/go-chi/chi/v5"` import used by that fragment. Create `cmd/middleware-api/config/base.yaml`:

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

The [distributed configuration](../../../../examples/chi-middleware/config/base.yaml) matches this file. Middleware is Go code. There is no YAML list of middleware names.

## 2. Keep middleware registration inside the application route group

Replace the basic route setup with the following fragment, using `net/http`, Bricks `response`, Bricks server `chi`, and the router alias `chiv5 "github.com/go-chi/chi/v5"`. Keep the existing `newGreetingRoutes` constructor. It appears below only for context, so do not add it twice:

```go
func exampleHeader(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Example", "bricks")
        next.ServeHTTP(w, r)
    })
}

func (r *GreetingRoutes) Setup(server *chi.Server) {
    server.Router().Group(func(router chiv5.Router) {
        router.Use(exampleHeader)
        router.Get("/hello", func(w http.ResponseWriter, request *http.Request) {
            if err := response.JSON(w, http.StatusOK, map[string]string{"message": "hello"}, nil); err != nil {
                slog.ErrorContext(request.Context(), "write greeting", "error", err)
            }
        })
    })
}

func newGreetingRoutes() *GreetingRoutes {
    return &GreetingRoutes{}
}
```

The base application contributes `newGreetingRoutes` to the Fx group named `routes` through `fx.As(new(chi.Route))` and `fx.ResultTags` annotations. Bricks calls `Setup` before serving. The group installs its middleware before its own routes. For authentication, replace the header middleware with your application authenticator and keep rejection responses within that middleware.

`chi.New` has already registered `/healthz` before application routes run. Calling `server.Router().Use(...)` on that root router at this point can panic because Chi requires middleware installation before root routes. Use `Group`, `With`, or a mounted subrouter. Do not recommend root `Use` from an Fx route's `Setup`.

## 3. Start and inspect both routes

```sh
APP_CONFIG_DIR=cmd/middleware-api/config APP_ENV=local go run ./cmd/middleware-api
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/hello
curl -i http://127.0.0.1:8080/healthz
```

Expect HTTP 200 with `X-Example: bricks` and `{"data":{"message":"hello"}}` from `/hello`. Expect HTTP 200 with `ok` and no `X-Example` header from `/healthz`. Press Ctrl-C to let the Fx runner drain the listener and preserve cleanup errors.

## Built-in and optional middleware

Bricks installs `middleware.RequestID`, then go-chi's `middleware.Logger` when `enablerequestlogging` is true, then `middleware.Recoverer`, then CORS when configured. Recovery is a final safety measure, not ordinary application error handling. Return expected errors to the application's response layer instead.

Bricks does not install `RealIP`, despite the older README's claim. If your deployment requires it, add `middleware.RealIP` to the application group before other group middleware. Import `github.com/go-chi/chi/v5/middleware`. That middleware trusts forwarding headers; only use it behind a proxy that removes untrusted incoming values. This group-level change does not affect built-in health or Swagger routes.

For browser policy rather than application middleware, use [CORS](cors.md). For built-in request logging and slog differences, use [logging](logging.md).
