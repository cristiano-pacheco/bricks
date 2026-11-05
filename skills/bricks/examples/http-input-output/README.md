# Run the Fx HTTP input and output example

This example installs the real request helpers, validator, response module, and Chi router behind a listening `net/http` server. It does not install the Bricks Chi server module. No YAML is required because all example settings are Go values.

## 1. Copy the application

Copy [main.go](main.go) to `cmd/http-input-output/main.go` in a Go module that already depends on Bricks, Fx, and Chi. Keep the application's selected Bricks dependency. The file contains every provider and lifecycle hook it needs.

`validator.Module` supplies the validator, `response.Module` supplies the error-handler interface, and `provideLogger` supplies `logger.Logger`. The demo logger disables its output sinks and flushes on stop. The server reports transport write errors through `slog`. Configure a production logger separately if you need persistent logs.

## 2. Start the listener

```sh
go run ./cmd/http-input-output
```

Expect `HTTP listening on http://127.0.0.1:8080`. Run these commands in another terminal:

```sh
curl -i -H 'Content-Type: application/json' --data '{"name":"Ada"}' http://127.0.0.1:8080/users
curl -i -H 'Content-Type: application/json' --data '{}' http://127.0.0.1:8080/decode
curl -i -H 'Content-Type: application/json' --data '{}' http://127.0.0.1:8080/users
curl -i -H 'Content-Type: application/json' --data '{"name":"abcdefghijklmnopqrstuvwxyz"}' http://127.0.0.1:8080/small
curl -i http://127.0.0.1:8080/users/42
curl -i http://127.0.0.1:8080/enveloped
curl -i http://127.0.0.1:8080/raw
curl -i -X DELETE http://127.0.0.1:8080/users/42
curl -i http://127.0.0.1:8080/structured-error
curl -i http://127.0.0.1:8080/unknown-error
```

Expected results below come from source inspection. This example has not been compiled or executed during this documentation change.

| Request | Status | Result |
| --- | --- | --- |
| Valid `POST /users` | 201 | `{"data":{"name":"Ada"}}` |
| Empty object at `/decode` | 201 | `{"data":{"name":""}}` |
| Empty object at `/users` | 422 | `INVALID_ARGUMENT`, with an English translated required-name detail |
| Large body at `/small` | 413 | `REQUEST_ENTITY_TOO_LARGE`, with a 32-byte limit message |
| `GET /users/42` | 200 | `{"data":{"id":"42"}}` |
| `/enveloped` | 200 | `{"data":{"name":"Ada"}}`, plus `X-Example: bricks` |
| `/raw` | 200 | `{"name":"Ada"}` |
| `DELETE /users/42` | 204 | Empty body |
| `/structured-error` | 404 | `USER_NOT_FOUND` and `user not found` inside `error` |
| `/unknown-error` | 500 | Generic `internal_server_error`, with no private message |

`/decode` and `/users` use the default 1,048,576-byte decode limit. Only `/users` calls the use case, whose first action is validation. `/small` demonstrates a 32-byte limit.

## 3. Stop and handle failures

Press Ctrl-C. Fx stops the HTTP server before flushing the logger. The application allows five seconds for shutdown, forces listener closure if graceful shutdown fails, and preserves server and cleanup errors. Unexpected serving failure requests Fx shutdown. `run` returns errors to `main`, which reports them only after cleanup and exits nonzero.

The focused [JSON guide](../../references/http/request/usage.md), [parameter guide](../../references/http/server/chi/request/usage.md), and [response guide](../../references/http/response/usage.md) each include a smaller standalone direct-construction program. Read only the guide for the behavior you need.
