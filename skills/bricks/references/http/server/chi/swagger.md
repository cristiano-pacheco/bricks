# Serve a Swagger UI and document

Serve Swagger UI at `/swagger/index.html` with a small document for `GET /hello`. Enabling the UI does not create a Swagger document or infer one from registered routes.

## 1. Copy the complete Swagger application

Copy [main.go](../../../../examples/chi-swagger/main.go) to `cmd/swagger-api/main.go` in your application module. Create `cmd/swagger-api/config/base.yaml`:

```yaml
app:
  http:
    address: 127.0.0.1:8080
    enablemetrics: false
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
    swagger:
      enabled: true
      path: /swagger/*
```

The [distributed configuration](../../../../examples/chi-swagger/config/base.yaml) matches this file. The application contains the same greeting route and cleanup-owning Fx runner as the basic server, plus document registration before constructing the app.

## 2. Register the document in Go

The copied program imports `github.com/swaggo/swag` and includes:

```go
type swaggerDocument struct{}

func (swaggerDocument) ReadDoc() string {
    return `{"swagger":"2.0","info":{"title":"Greeting API","version":"1.0"},"basePath":"/","paths":{"/hello":{"get":{"responses":{"200":{"description":"Greeting"}}}}}}`
}

func application() *fx.App {
    swag.Register("swagger", swaggerDocument{})
    return fx.New(
        chi.Module,
        fx.Provide(fx.Annotate(newGreetingRoutes, fx.ResultTags(`group:"routes"`))),
    )
}
```

This is the implementation already present in the copied program. The default document instance is `swagger`. Register it once per process. Duplicate registration may panic in the upstream library, so do not call this application constructor repeatedly or combine it with a generated package that registers the same instance.

## 3. Run and request the UI and document

```sh
APP_CONFIG_DIR=cmd/swagger-api/config APP_ENV=local go run ./cmd/swagger-api
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/swagger/index.html
curl -i http://127.0.0.1:8080/swagger/doc.json
curl -i http://127.0.0.1:8080/hello
```

Expect HTTP 200 HTML from the UI, HTTP 200 JSON with title `Greeting API` from `doc.json`, and HTTP 200 with `{"data":{"message":"hello"}}` from the application route. Open the index URL in a browser for the interactive UI. Press Ctrl-C to stop through the Fx runner and preserve shutdown errors.

## Supported alternatives and limitations

For generated documentation, replace the manual registration with a blank import of your application's generated docs package, such as `_ "example.com/myapp/docs"`. That package must register the default document instance. Bricks does not run the generator for you.

An omitted or null `swagger` disables the route. `enabled: false` also disables it. Empty `path` uses `/swagger/*`; a nonempty path without `/*` gets that suffix appended. Use an absolute route prefix. These settings affect the route, not document registration.

`SwaggerConfig` exports only `Enabled` and `Path`. There is no `dir` YAML setting. A stale source comment describes a directory, but the implementation does not read one. The built-in handler has no Bricks YAML contract for custom handler options, security, or a custom document instance name. For those cases, disable built-in Swagger and mount an application-owned handler with the appropriate options.

`SetupRoutes()` installs the Swagger route. Fx calls it automatically; direct callers must call it before `Start()`. Do not call it repeatedly after serving starts. Swagger does not add authentication. Restrict exposure using your deployment's routing policy or an application-owned secured mount.
