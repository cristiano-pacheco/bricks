# Read a Chi path parameter

Build `GET /users/{id}` and return the captured ID as a JSON string. The application needs Bricks and `github.com/go-chi/chi/v5` as dependencies. This helper has no YAML configuration, constructor, Fx module, validation, or shutdown hook.

## 1. Register the route

Create `cmd/parameter-demo/main.go`:

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

	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	pathrequest "github.com/cristiano-pacheco/bricks/pkg/http/server/chi/request"
	"github.com/go-chi/chi/v5"
)

func run() error {
	router := chi.NewRouter()
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := pathrequest.Param(r, "id")
		if err := response.JSON(w, http.StatusOK, map[string]string{"id": id}, nil); err != nil {
			slog.Error("write user response", "error", err)
		}
	})
	return serve(router)
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

`Param` reads Chi's route context. The route must use `{id}` and the handler must ask for the same name. The helper does not read a query parameter or JSON field.

## 2. Run and request an ID

Run `go run ./cmd/parameter-demo`, then:

```sh
curl -i http://127.0.0.1:8080/users/42
curl -i http://127.0.0.1:8080/users/abc
curl -i 'http://127.0.0.1:8080/users/42?id=99'
```

Expect 200 with `{"data":{"id":"42"}}`, `{"data":{"id":"abc"}}`, and `{"data":{"id":"42"}}` respectively. These are source-derived expectations, not executed results.

## 3. Stop the server

Press Ctrl-C. The listener drains requests for up to five seconds, closes remaining connections if needed, and returns cleanup errors before exiting.

## Missing values and typed IDs

`Param(r, name)` is a thin wrapper around `chi.URLParam`. An absent route parameter or missing Chi context returns an empty string. A request to `/users` does not match `/users/{id}` at all, so Chi returns 404 before this handler runs.

Parse numeric IDs explicitly with `strconv.ParseInt` and handle its error before calling the use case. `abc` is a valid string capture, not a numeric validation error from this package. Do not trust an ID merely because the router matched it. Authorization belongs to application logic.

## Fx alternative

The helper needs no Fx provider. Call `Param` inside the route handler, as in the complete inline application above. For the Bricks Chi server's route groups and automatic lifecycle integration, load [Chi server usage](../usage.md) only when installing that server. For body decoding, load [JSON requests](../../../request/usage.md) only when the route has a JSON body.
