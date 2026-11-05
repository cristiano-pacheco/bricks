# Allow a browser origin

Allow a browser at `https://app.example.com` to call your Chi application with credentials. This adds CORS response headers. It does not authenticate requests or protect routes from non-browser clients.

## 1. Copy the application and configuration

In your application module, create `cmd/browser-api/main.go` from the complete inline application in [server usage](usage.md). It includes the Fx runner and a `chi.Route` implementation for `GET /hello`. Create `cmd/browser-api/config/base.yaml`:

```yaml
app:
  http:
    address: 127.0.0.1:8080
    enablemetrics: false
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
    cors:
      allowedorigins: ["https://app.example.com"]
      allowedmethods: ["GET", "POST", "OPTIONS"]
      allowedheaders: ["Accept", "Authorization", "Content-Type"]
      exposedheaders: ["X-Example"]
      allowcredentials: true
      maxage: 300
      optionspassthrough: false
      debug: false
```

The distributed [minimal file](../../../../examples/chi-basic/config/base.yaml) plus [CORS overlay](../../../../examples/chi-basic/config/cors.yaml) contain these same values. CORS comes from `chi.New` before routes are installed. No extra Fx module or application middleware is needed.

The copied route is ordinary Go, not a YAML declaration:

```go
func (r *GreetingRoutes) Setup(server *chi.Server) {
    server.Router().Get("/hello", func(w http.ResponseWriter, request *http.Request) {
        if err := response.JSON(w, http.StatusOK, map[string]string{"message": "hello"}, nil); err != nil {
            slog.ErrorContext(request.Context(), "write greeting", "error", err)
        }
    })
}
```

This is the route from the basic inline application, not an additional replacement file. It uses `net/http`, `log/slog`, and the Bricks `response` and `chi` imports already present there.

## 2. Run and send a browser-style request

```sh
APP_CONFIG_DIR=cmd/browser-api/config APP_ENV=local go run ./cmd/browser-api
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/hello \
  -H 'Origin: https://app.example.com'
curl -i -X OPTIONS http://127.0.0.1:8080/hello \
  -H 'Origin: https://app.example.com' \
  -H 'Access-Control-Request-Method: GET' \
  -H 'Access-Control-Request-Headers: Authorization'
```

Expect HTTP 200 and `{"data":{"message":"hello"}}` for GET, with `Access-Control-Allow-Origin: https://app.example.com` and `Access-Control-Allow-Credentials: true`. The preflight returns an empty HTTP 200 response with CORS allow headers and `Access-Control-Max-Age: 300`.

A disallowed origin does not receive an allow-origin header. That does not necessarily reject the server-side operation, because browsers enforce CORS. Keep authentication and authorization in application middleware or use cases.

## 3. Stop the application

Press Ctrl-C. The copied runner stops the Fx application and reports cleanup errors at `main` after Chi drains its listener.

## Supported alternatives

For an unauthenticated public browser API, use a separate scenario with `allowedorigins: ["*"]` and `allowcredentials: false`. Do not copy a wildcard into the credentialed scenario above. Omit `cors`, or use `cors: null`, to install no CORS middleware.

For direct construction, assign `cfg.CORS = &chi.CORSConfig{...}` before `chi.New(cfg)`. `cfg = cfg.WithDefaultCORS()` selects Bricks' explicit permissive defaults. It is not a method on the server.

All nested fields, zero values, and upstream defaults are in [the configuration reference](configuration.md#cors-fields). If `optionspassthrough` is true, add an application OPTIONS handler where needed. The handler must finish the preflight response rather than relying on CORS to stop the request.
