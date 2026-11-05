# Run an explicitly decorated HTTP use case

Use this complete application in a Go module that already depends on Bricks. The [usage guide](../../recipes/http-flow.md) explains the wiring and alternatives.

## 1. Copy the files

Copy this entire directory to `cmd/http-flow` in your application. Keep `main.go` and `config/base.yaml` together. No repository-relative source or other skill files are needed to run it.

## 2. Run from your application root

```sh
APP_CONFIG_DIR=cmd/http-flow/config APP_ENV=local go run ./cmd/http-flow
```

## 3. Send the request in another terminal

```sh
curl -i http://127.0.0.1:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"sku":"coffee","quantity":2}'
```

Expect HTTP 201 with `{"data":{"total_cents":500}}`.

Use quantity zero for a validation 422, SKU `missing` for a structured 404, and SKU `broken` for a generic 500. The catalog is an in-memory application adapter, not persisted storage. Manual decoration logs to `orders.log`; metrics and translation are omitted. The tracing wrapper has no exporter installed.

## 4. Stop

Press Ctrl-C. The Fx runner waits for server shutdown and joins lifecycle errors before reporting failure at `main`. The HTTP-flow example also synchronizes its application-owned logger files.

Execution, compilation, YAML loading, and graph checks were waived for this documentation task. These expected results are source-backed, not observed runtime results.
