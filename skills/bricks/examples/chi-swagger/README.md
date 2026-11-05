# Serve Swagger UI with a registered document

Use this complete application in a Go module that already depends on Bricks. The [usage guide](../../references/http/server/chi/swagger.md) explains the wiring and alternatives.

## 1. Copy the files

Copy this entire directory to `cmd/chi-swagger` in your application. Keep `main.go` and `config/base.yaml` together. No repository-relative source or other skill files are needed to run it.

## 2. Run from your application root

```sh
APP_CONFIG_DIR=cmd/chi-swagger/config APP_ENV=local go run ./cmd/chi-swagger
```

## 3. Send the request in another terminal

```sh
curl -i http://127.0.0.1:8080/swagger/index.html
curl -i http://127.0.0.1:8080/swagger/doc.json
```

Expect HTTP 200 HTML for the UI and HTTP 200 JSON with document title `Greeting API`.

The program registers one `swag` document before Fx construction. Do not also import a generated package that registers the same default instance. No generator or framework checkout is required.

## 4. Stop

Press Ctrl-C. The Fx runner waits for server shutdown and joins lifecycle errors before reporting failure at `main`. The HTTP-flow example also synchronizes its application-owned logger files.

Execution, compilation, YAML loading, and graph checks were waived for this documentation task. These expected results are source-backed, not observed runtime results.
