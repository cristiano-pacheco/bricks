# Add middleware to application routes

Use this complete application in a Go module that already depends on Bricks. The [usage guide](../../references/http/server/chi/middleware.md) explains the wiring and alternatives.

## 1. Copy the files

Copy this entire directory to `cmd/chi-middleware` in your application. Keep `main.go` and `config/base.yaml` together. No repository-relative source or other skill files are needed to run it.

## 2. Run from your application root

```sh
APP_CONFIG_DIR=cmd/chi-middleware/config APP_ENV=local go run ./cmd/chi-middleware
```

## 3. Send the request in another terminal

```sh
curl -i http://127.0.0.1:8080/hello
curl -i http://127.0.0.1:8080/healthz
```

Expect HTTP 200 with the greeting envelope and `X-Example: bricks` on `/hello`. The health route has no custom header.

Middleware belongs to the application group. Root middleware cannot be added after Bricks has registered its health route.

## 4. Stop

Press Ctrl-C. The Fx runner waits for server shutdown and joins lifecycle errors before reporting failure at `main`. The HTTP-flow example also synchronizes its application-owned logger files.

Execution, compilation, YAML loading, and graph checks were waived for this documentation task. These expected results are source-backed, not observed runtime results.
