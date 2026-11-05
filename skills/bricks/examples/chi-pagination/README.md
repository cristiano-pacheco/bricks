# Compose pagination with HTTP response metadata

Use this complete application in a Go module that already depends on Bricks. The [usage guide](../../recipes/pagination-response.md) explains the wiring and alternatives.

## 1. Copy the files

Copy this entire directory to `cmd/chi-pagination` in your application. Keep `main.go` and `config/base.yaml` together. No repository-relative source or other skill files are needed to run it.

## 2. Run from your application root

```sh
APP_CONFIG_DIR=cmd/chi-pagination/config APP_ENV=local go run ./cmd/chi-pagination
```

## 3. Send the request in another terminal

```sh
curl -i 'http://127.0.0.1:8080/products?page=2&per_page=20'
```

Expect HTTP 200 with `{"data":{"items":["cocoa"],"pagination":{"total_count":3,"page":2,"per_page":2,"total_pages":2}}}`.

The strict parser rejects invalid supplied values. Application normalization caps the size at two. A separate page-number bound protects offset multiplication. No YAML pagination policy exists.

## 4. Stop

Press Ctrl-C. The Fx runner waits for server shutdown and joins lifecycle errors before reporting failure at `main`. The HTTP-flow example also synchronizes its application-owned logger files.

Execution, compilation, YAML loading, and graph checks were waived for this documentation task. These expected results are source-backed, not observed runtime results.
