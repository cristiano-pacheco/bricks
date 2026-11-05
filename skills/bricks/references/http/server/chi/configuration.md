# Configure the Chi server

Choose listener addresses and request deadlines in `config/base.yaml`. This guide covers the complete `chi.Config` YAML contract. Route implementations, middleware functions, logger interfaces, and Swagger document registration stay in Go.

## 1. Start with a single listener

Use the [server application's complete inline program](usage.md) or copy [its main.go](../../../../examples/chi-basic/main.go) to `cmd/hello/main.go`. Create `cmd/hello/config/base.yaml`:

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

This configuration listens only on loopback and explicitly disables metrics. `chi.Module` loads the `app.http` subtree using `config.Provide[chi.Config]("app.http")`. The file must exist and the subtree must be present. `APP_CONFIG_DIR` selects the directory, default `config`. `APP_ENV` selects an overlay, default `local`; a missing overlay is optional. For overlays and `env://` references beyond the scenarios below, load [configuration usage](../../../config/usage.md).

## 2. Run and inspect the result

```sh
APP_CONFIG_DIR=cmd/hello/config APP_ENV=local go run ./cmd/hello
curl -i http://127.0.0.1:8080/hello
```

Run curl in another terminal. Expect HTTP 200 and `{"data":{"message":"hello"}}`. Stop with Ctrl-C. Fx owns the listeners and the application runner waits for shutdown.

## 3. Copy the full-field example when you need those settings

The following is [full.yaml](../../../../examples/chi-basic/config/full.yaml). Keep the minimal `base.yaml`, copy this file beside it, and run with `APP_ENV=full`. It enables a second listener and a specific browser origin. Swagger remains disabled until the application registers a document as explained in [its branch](swagger.md).

```yaml
app:
  http:
    address: 127.0.0.1:8080
    port: 8080
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
    enablemetrics: true
    metricsaddress: 127.0.0.1:9091
    metricsport: 9091
    enablerequestlogging: true
    enableroutelogging: true
    cors:
      allowedorigins: ["https://app.example.com"]
      allowedmethods: ["GET", "POST", "OPTIONS"]
      allowedheaders: ["Accept", "Authorization", "Content-Type"]
      exposedheaders: ["X-Example"]
      allowcredentials: true
      maxage: 300
      optionspassthrough: false
      debug: false
    swagger:
      enabled: false
      path: /swagger/*
```

Every field below is optional in the type, but an omitted address and port select an ephemeral wildcard listener, not port 8080. No field in the YAML loader receives `chi.Default()` automatically. Explicit address values take precedence over the corresponding port fields.

## Listener and lifecycle fields

| Key under `app.http` | Type | Omitted or zero behavior | Direct `chi.Default()` | Constraints |
| --- | --- | --- | --- | --- |
| `address` | string | Empty uses `0.0.0.0:port` | Empty | Nonempty must be `host:port`; surrounding whitespace is trimmed. IPv6 needs brackets. |
| `port` | uint | `0` requests an available port | `8080` | Effective numeric port must be 0 through 65535. Ignored when `address` is nonempty. |
| `readtimeout` | duration string | `0` means no read deadline | `15s` | Go duration units, for example `500ms` or `1m`; forwarded to both enabled listeners. |
| `writetimeout` | duration string | `0` means no write deadline | `15s` | Go duration units; forwarded to both enabled listeners. |
| `idletimeout` | duration string | `0` lets net/http fall back to ReadTimeout | `60s` | Negative durations disable the idle deadline in net/http; the config validator does not reject them. |
| `shutdowntimeout` | duration string | `0` adds no internal Fx shutdown deadline | `10s` | Positive values bound Chi's Fx stop hook; nonpositive values leave the caller's context unchanged. Direct Shutdown uses its passed context. |
| `enablemetrics` | bool | `false`, no metrics listener | `true` | Independent of any `app.metrics` configuration. |
| `metricsaddress` | string | Empty uses `0.0.0.0:metricsport` | Empty | Same address format as `address`; validated only when metrics are enabled. |
| `metricsport` | uint | `0` requests an available port | `9090` | Ignored when `metricsaddress` is nonempty. Nonzero effective port must differ from the public listener's, regardless of host. |
| `enablerequestlogging` | bool | `false` | `true` | Enables go-chi's text request logging middleware, not Bricks' logger. |
| `enableroutelogging` | bool | `false` | `true` | Enables startup address and route messages through slog. |
| `cors` | object or null | `nil`, no CORS middleware | `nil` | An empty object enables middleware with upstream zero-value option behavior. |
| `swagger` | object or null | `nil`, no Swagger route | `nil` | UI is installed during SetupRoutes only when `enabled` is true. |

Durations are code `time.Duration` values after loading. Validation checks addresses and listener port conflicts, not sensible timeout policy. Set positive deadlines explicitly for your deployment. Hostnames are passed to TCP binding; invalid or unavailable hosts fail at startup.

## CORS fields

The fields below are under `app.http.cors`. None is required by the decoder. See [the copyable browser scenario](cors.md) before enabling credentials.

| Key | Type | Zero behavior and constraints |
| --- | --- | --- |
| `allowedorigins` | list of strings | Empty lets go-chi/cors use its wildcard origin default. Use explicit browser origins when credentials are enabled. |
| `allowedmethods` | list of strings | Empty uses upstream GET, POST, HEAD defaults. Methods must match actual browser requests. |
| `allowedheaders` | list of strings | Empty uses upstream Origin, Accept, Content-Type defaults. Add Authorization when required. |
| `exposedheaders` | list of strings | Empty exposes no additional headers to browser JavaScript. |
| `allowcredentials` | bool | `false`; true permits credentialed CORS responses. Avoid wildcard origins for credentialed requests. |
| `maxage` | int | `0` omits preflight cache duration. Positive values are seconds, not duration strings. |
| `optionspassthrough` | bool | `false`, middleware answers preflight; true lets the next handler receive OPTIONS. |
| `debug` | bool | `false`; true enables upstream CORS debug logging. |

`Config.WithDefaultCORS()` returns a modified copy with origins `*`, methods GET/POST/PUT/PATCH/DELETE/OPTIONS, headers Accept/Authorization/Content-Type, credentials false, and max age 300 seconds. That explicit helper differs from a YAML empty CORS object.

## Swagger fields

| Key under `app.http.swagger` | Type | Zero behavior and constraints |
| --- | --- | --- |
| `enabled` | bool | `false`; true installs the UI route during SetupRoutes. |
| `path` | string | Empty uses `/swagger/*`. A nonempty path gets `/*` appended unless it already has that suffix. Use an absolute Chi route pattern. |

There is no `dir` field in `SwaggerConfig`, despite the stale Go comment mentioning one. Supply the Swagger document through Go registration or an imported generated docs package. YAML does not define document files.

## Go-only options and direct defaults

Direct construction can start with `cfg := chi.Default()`, then `server, err := chi.New(cfg)`. `New` accepts a Config, not variadic options. Apply an exported `Option` explicitly to `&cfg` when desired:

```go
// Fragment before chi.New(cfg).
chi.WithAddress("127.0.0.1:8080")(&cfg)
chi.WithMetrics(false)(&cfg)
```

`WithPort` clears `Address`; `WithMetricsPort` clears `MetricsAddress`. The address setters replace addresses without changing fallback port fields. `WithReadTimeout`, `WithWriteTimeout`, `WithIdleTimeout`, and `WithShutdownTimeout` set durations. `WithRequestLogging`, `WithRouteLogging`, and `WithMetrics` set flags. `WithCORS`, `WithDefaultCORS`, and `WithSwagger` replace their corresponding nested configuration. These functions do not mutate a constructed server.

For the port-only alternative, keep the minimal base file and copy [ports.yaml](../../../../examples/chi-basic/config/ports.yaml) beside it. Run with `APP_ENV=ports`:

```yaml
app:
  http:
    address: ""
    port: 8080
    enablemetrics: true
    metricsaddress: ""
    metricsport: 9091
```

This clears the explicit addresses inherited from the base file and binds both listeners on `0.0.0.0`. Use that wildcard exposure only when intended.

For ephemeral public and metrics listeners, copy [ephemeral.yaml](../../../../examples/chi-basic/config/ephemeral.yaml) beside the minimal file and run with `APP_ENV=ephemeral`. Both explicit addresses are `127.0.0.1:0`. Read effective addresses from startup route logs or `Server.Addr()` and `Server.MetricsAddr()`. Configuration loading alone cannot prove successful binds or HTTP behavior.
