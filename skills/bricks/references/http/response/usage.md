# Choose an HTTP response contract

Build independent routes for enveloped JSON, raw JSON, empty success, and mapped errors. Bricks must already be an application dependency. Response behavior is code-only. There are no YAML keys for envelopes, status codes, headers, validators, or error-handler dependencies.

## 1. Create the response application

Create `cmd/response-demo/main.go`:

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

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/validator"
)

func run() error {
	v, err := validator.New()
	if err != nil {
		return err
	}
	// Nil dependencies are a supported direct-construction fallback.
	// Validation messages below use the handler's deterministic fallback format.
	h := response.NewErrorHandler(nil, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /enveloped", func(w http.ResponseWriter, r *http.Request) {
		headers := http.Header{"X-Example": {"bricks"}}
		if err := response.JSON(w, http.StatusOK, map[string]string{"name": "Ada"}, headers); err != nil {
			slog.Error("write enveloped response", "error", err)
		}
	})
	mux.HandleFunc("GET /raw", func(w http.ResponseWriter, r *http.Request) {
		if err := response.JSONRaw(w, http.StatusOK, map[string]string{"name": "Ada"}, nil); err != nil {
			slog.Error("write raw response", "error", err)
		}
	})
	mux.HandleFunc("DELETE /users/42", func(w http.ResponseWriter, r *http.Request) {
		response.NoContent(w)
	})
	mux.HandleFunc("GET /structured-error", func(w http.ResponseWriter, r *http.Request) {
		h.Error(w, errs.New("USER_NOT_FOUND", "user not found", http.StatusNotFound, nil))
	})
	mux.HandleFunc("GET /unknown-error", func(w http.ResponseWriter, r *http.Request) {
		h.Error(w, errors.New("private internal detail"))
	})
	mux.HandleFunc("GET /validation-error", func(w http.ResponseWriter, r *http.Request) {
		input := struct { Name string `validate:"required"` }{}
		if err := v.Validate(input); err != nil {
			h.Error(w, err)
			return
		}
		response.NoContent(w)
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

Success functions return marshal or write errors, so the handlers report those errors without attempting another response after headers might have been sent. The error handler writes errors itself and logs its own marshal or write failures. `NoContent` has no error return.

The direct handler deliberately receives nil validator and logger dependencies. It maps validation failures with fallback field messages and uses `slog.Default()` for internal response-write failures. The separate validator still produces validation errors.

## 2. Run and select the contract

Run `go run ./cmd/response-demo`. In another terminal:

```sh
curl -i http://127.0.0.1:8080/enveloped
curl -i http://127.0.0.1:8080/raw
curl -i -X DELETE http://127.0.0.1:8080/users/42
curl -i http://127.0.0.1:8080/structured-error
curl -i http://127.0.0.1:8080/unknown-error
curl -i http://127.0.0.1:8080/validation-error
```

Expected results, derived from source and not executed:

| Route | Status | Body |
| --- | --- | --- |
| `/enveloped` | 200 | `{"data":{"name":"Ada"}}` |
| `/raw` | 200 | `{"name":"Ada"}` |
| `DELETE /users/42` | 204 | No body |
| `/structured-error` | 404 | `{"error":{"code":"USER_NOT_FOUND","message":"user not found"}}` |
| `/unknown-error` | 500 | `{"error":{"code":"internal_server_error","message":"Internal server error"}}` |
| `/validation-error` | 422 | `{"error":{"code":"INVALID_ARGUMENT","message":"request has invalid fields","details":[{"field":"name","message":"name: required"}]}}` |

Object key order is not part of the contract. The enveloped response also has `X-Example: bricks`. Both JSON success routes set `Content-Type: application/json` and a byte `Content-Length`.

## 3. Shut down

Press Ctrl-C. The server allows five seconds to drain requests and joins cleanup errors before reporting failure at `main`. The response helpers and validator own no background process.

## Fx with translated validation messages

Prefer `response.Module` in an Fx application that already uses dependency injection. Follow the inline [HTTP flow](../../../recipes/http-flow.md) for the complete graph and listener. For an application that uses `logger.Module`, the dependency fragment is:

```go
// Dependency fragment inside fx.New.
validator.Module,
logger.Module,
response.Module,
```

`response.Module` provides `response.ErrorHandler`, not a standalone `response.Error` function. It requires `validator.Validator` and `logger.Logger`. Import `github.com/cristiano-pacheco/bricks/pkg/logger` for `logger.Module`, which also requires real `app.logger` configuration. The HTTP-flow recipe instead constructs its logger in Go and includes route providers and the runner. Choose one logger provider, not both. The fragment above supplies only dependencies, not a runnable application.

With a validator supplied to `NewErrorHandler`, validation details use its translator. Without one they use `snake_case_field: tag`. Direct construction returns `*response.ErrorHandlerImpl`, which implements `response.ErrorHandler`; no lifecycle hook is needed for the handler itself.

## Functions and status mapping

- `JSON(w, status, data, headers)` marshals `NewEnvelope(data)`, a map with one `data` key. `JSONRaw` marshals `data` directly. Both marshal the full body before writing headers. They do not stream JSON and make no zero-allocation promise.
- `Envelope` is `map[string]any`. If you build a custom top-level envelope with metadata, pass it to `JSONRaw`; passing it to `JSON` adds another `data` wrapper.
- Both JSON writers set content type and content length, then add every supplied header value with `Header.Add`. Avoid supplying conflicting content-type or content-length headers. The caller chooses a valid HTTP status and a JSON-marshalable value.
- `NoContent` writes 204 only. It does not marshal JSON or clear headers already written by other code.
- Error mapping uses `errors.As` semantics. Wrapped validator errors become 422, before structured-error matching. Validation detail fields use the validator's Go field names converted to snake case, not necessarily JSON tags.
- `*errs.Error` preserves its code, message, details, and nonzero status. A zero status becomes 500 and mutates the error object. Avoid reusing mutable structured errors across requests. Only expose safe messages and details. `Status` and `OriginalError` do not appear in JSON.
- Unknown errors produce the generic 500 body above. Their original message is not included, and the handler does not automatically log every input error. Log operational failures in application code when needed. Passing a nil error also takes this generic path. Do not pass an error interface holding a nil `*errs.Error`; the structured-error branch dereferences that pointer.
- Error responses set content type but do not explicitly set content length. Marshal failure falls back to a generic 500. Write failures can no longer change a status already sent.

For decoding failures, load [JSON request usage](../request/usage.md). For validation rules or translator extension, load [validator usage](../../validator/usage.md). For structured-error construction and original-error behavior, load [errors usage](../../errs/usage.md). For a complete decorated HTTP use case, select the HTTP-flow recipe only when that composition is requested.
