# Run a Chi greeting server

Use this complete application in a Go module that already depends on Bricks. The [usage guide](../../references/http/server/chi/usage.md) explains the wiring and alternatives.

## 1. Copy the files

Copy this entire directory to `cmd/chi-basic` in your application. Keep `main.go` and `config/base.yaml` together. No repository-relative source or other skill files are needed to run it.

## 2. Run from your application root

```sh
APP_CONFIG_DIR=cmd/chi-basic/config APP_ENV=local go run ./cmd/chi-basic
```

## 3. Send the request in another terminal

```sh
curl -i http://127.0.0.1:8080/hello
```

Expect HTTP 200 with `{"data":{"message":"hello"}}`. `/healthz` returns `ok`.

The minimal YAML disables metrics. Optional overlay files keep CORS, logging, metrics, Swagger, and ephemeral addresses separate. For example, keep `base.yaml` and select `APP_ENV=cors` for the CORS overlay or `APP_ENV=full` for every field. Swagger also needs document registration, so use the separate `chi-swagger` example rather than enabling its flag alone.

## 4. Stop

Press Ctrl-C. The Fx runner waits for server shutdown and joins lifecycle errors before reporting failure at `main`. The HTTP-flow example also synchronizes its application-owned logger files.

Execution, compilation, YAML loading, and graph checks were waived for this documentation task. These expected results are source-backed, not observed runtime results.
