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
			slog.ErrorContext(r.Context(), "write user response", "error", err)
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
