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
			slog.ErrorContext(r.Context(), "write enveloped response", "error", err)
		}
	})
	mux.HandleFunc("GET /raw", func(w http.ResponseWriter, r *http.Request) {
		if err := response.JSONRaw(w, http.StatusOK, map[string]string{"name": "Ada"}, nil); err != nil {
			slog.ErrorContext(r.Context(), "write raw response", "error", err)
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
		input := struct {
			Name string `validate:"required"`
		}{}
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
