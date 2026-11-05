# Decode a JSON request

Build `POST /users`, reject unsupported media types and unknown fields, and return the decoded input. Bricks must already be a dependency of your Go application. These helpers have no Fx module, YAML configuration, or owned resources. The application owns the listener.

## 1. Create the application

Create `cmd/json-demo/main.go` with the code below. The 32-byte limit makes the size error easy to reproduce. `validate:"required"` does not run during decoding, so an empty name still succeeds here.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/request"
	"github.com/cristiano-pacheco/bricks/pkg/http/response"
)

type Input struct {
	Name string `json:"name" validate:"required"`
}

func run() error {
	h := response.NewErrorHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var input Input
		if err := request.ReadJSONWithMaxSize(w, r, &input, 32); err != nil {
			h.Error(w, err)
			return
		}
		if err := response.JSON(w, http.StatusCreated, input, nil); err != nil {
			// Headers may already be sent. Report the write error without a second response.
			slog.Error("write user response", "error", err)
		}
	})
	return serve(mux)
}

func serve(handler http.Handler) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fmt.Println("HTTP listening on http://127.0.0.1:8080")
	select {
	case err = <-done:
		return errors.Join(err, server.Close())
	case <-ctx.Done():
	}
	stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	err = server.Shutdown(stopCtx)
	if err != nil {
		err = errors.Join(err, server.Close())
	}
	serveErr := <-done
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(err, serveErr)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

The response handler accepts nil validator and logger dependencies for direct construction. It uses fallback validation messages and `slog.Default()` for response marshal or write failures. The example checks success-response write errors itself.

## 2. Start the server and send requests

Run `go run ./cmd/json-demo`. In another terminal:

```sh
curl -i -H 'Content-Type: application/json' --data '{"name":"Ada"}' http://127.0.0.1:8080/users
```

Expect 201 and `{"data":{"name":"Ada"}}`.

Use these independent requests to exercise the boundary:

```sh
# Missing Content-Type is accepted.
curl -i -H 'Content-Type:' --data '{"name":"Ada"}' http://127.0.0.1:8080/users
# Parameters after the semicolon are accepted.
curl -i -H 'Content-Type: application/json; charset=utf-8' --data '{"name":"Ada"}' http://127.0.0.1:8080/users
# Unsupported media type.
curl -i -H 'Content-Type: text/plain' --data '{}' http://127.0.0.1:8080/users
# Unknown field, malformed input, wrong type, empty input, and another value.
curl -i -H 'Content-Type: application/json' --data '{"admin":true}' http://127.0.0.1:8080/users
curl -i -H 'Content-Type: application/json' --data '{"name":}' http://127.0.0.1:8080/users
curl -i -H 'Content-Type: application/json' --data '{"name":1}' http://127.0.0.1:8080/users
curl -i -H 'Content-Type: application/json' --data '' http://127.0.0.1:8080/users
curl -i -H 'Content-Type: application/json' --data '{} {}' http://127.0.0.1:8080/users
# The JSON body exceeds 32 bytes.
curl -i -H 'Content-Type: application/json' --data '{"name":"abcdefghijklmnopqrstuvwxyz"}' http://127.0.0.1:8080/users
# Decoding is not validation.
curl -i -H 'Content-Type: application/json' --data '{}' http://127.0.0.1:8080/users
```

The first two return 201. Other expected bodies are listed below. These are source-derived expectations, not executed results.

| Input | Status | Error code | Error message |
| --- | --- | --- | --- |
| `text/plain` | 415 | `UNSUPPORTED_MEDIA_TYPE` | `Content-Type header is not application/json` |
| Unknown `admin` field | 400 | `BAD_REQUEST` | `request body contains unknown field "admin"` |
| Malformed JSON | 400 | `BAD_REQUEST` | `request body contains malformed JSON` |
| Numeric name | 400 | `BAD_REQUEST` | `request body contains invalid value for field "name"` |
| Empty body | 400 | `BAD_REQUEST` | `request body must not be empty` |
| Two objects | 400 | `BAD_REQUEST` | `request body must contain only a single JSON value` |
| Oversized name | 413 | `REQUEST_ENTITY_TOO_LARGE` | `request body must not exceed 32 bytes` |

Each error uses `{"error":{"code":"...","message":"..."}}`. The final `{}` returns 201 with `{"data":{"name":""}}`.

## 3. Stop the listener

Press Ctrl-C in the server terminal. The application allows five seconds for active requests, then closes remaining connections if shutdown fails. It joins shutdown and serving errors before reporting failure at `main`.

## Use the default limit or add validation

Replace the decode call with `request.ReadJSON(w, r, &input)` to use `DefaultMaxBodySize`, 1,048,576 bytes. `ReadJSONWithMaxSize` accepts an `int64` byte count and installs `http.MaxBytesReader`. Supply a positive limit appropriate for your endpoint. The helper does not validate the limit as application configuration.

For decoding followed by use-case validation and complete Fx response dependencies, follow [the HTTP flow](../../../recipes/http-flow.md). Its inline application provides the validator, response handler, logger, route contribution, and lifecycle ownership. The request helper itself needs no provider. For custom validation rules, load [validator usage](../../validator/usage.md) only when needed.

## Decoder behavior and limits

- The content-type check accepts an empty header. Otherwise it strips semicolon parameters, trims whitespace, and compares case-sensitively with `application/json`. It does not accept `application/problem+json` or uppercase variants.
- Unknown-field rejection applies to decoded structs, not arbitrary keys in maps. Decoding is not business validation, authentication, authorization, CSRF protection, or a guarantee against denial of service.
- `null` can decode successfully without populating a struct. Validate application input after decoding, preferably at the start of the use case.
- The implementation calls `Decoder.More()` after one decode. It rejects ordinary consecutive values but does not perform a second decode followed by an explicit EOF check. Do not claim a complete trailing-data or total-body-consumption guarantee. Size enforcement occurs as the decoder reads, not as an independent pre-scan of the whole body.
- Syntax and unexpected-EOF errors become malformed-JSON errors. Type errors identify a field when available. Other decode errors become `BAD_REQUEST` with `error parsing request body`.
- Pass a non-nil pointer to the destination. An invalid destination triggers the library's panic for `json.InvalidUnmarshalError`; this is a programming error, not a supported client-error response.

For response envelopes or error translation, load [response usage](../response/usage.md). Do not load server configuration or storage guides merely to decode JSON.
