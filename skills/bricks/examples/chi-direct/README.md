# Run Chi without Fx

This complete program owns direct server startup and shutdown. It needs an existing application module with Bricks, but no YAML configuration.

## 1. Copy the file

Copy `main.go` to `cmd/chi-direct/main.go`. The code starts from `chi.Default()`, selects `127.0.0.1:0`, and disables metrics. It registers `/hello` directly and calls `SetupRoutes()` once before serving.

## 2. Run and note the address

```sh
go run ./cmd/chi-direct
```

The program prints `HTTP listening on http://127.0.0.1:<port>`. Replace `<port>` in the next command with that value. The polling loop waits for binding or its error before shutdown, even when a signal arrives during startup.

## 3. Request and stop

```sh
curl -i http://127.0.0.1:<port>/hello
```

Expect HTTP 200 with `{"data":{"message":"hello"}}`. Press Ctrl-C. The program calls `Shutdown` with a deadline, waits for the serving result, and treats the expected `http.ErrServerClosed` as success. Other serving and cleanup errors reach `main` after cleanup.

The [server guide](../../references/http/server/chi/usage.md) explains Fx as the recommended lifecycle alternative. Compilation and runtime execution were waived for this task, so the result above remains unverified by execution.
