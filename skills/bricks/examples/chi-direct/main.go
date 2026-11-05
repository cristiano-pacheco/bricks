package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
)

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg := chi.Default()
	cfg.Address = "127.0.0.1:0"
	cfg.EnableMetrics = false
	server, err := chi.New(cfg)
	if err != nil {
		return err
	}
	server.Router().Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		if err := response.JSON(w, http.StatusOK, map[string]string{"message": "hello"}, nil); err != nil {
			slog.ErrorContext(r.Context(), "write greeting", "error", err)
		}
	})
	server.SetupRoutes()
	finished := make(chan error, 1)
	go func() { finished <- server.Start() }()
	// Addr changes after synchronous binding. Wait for that change or a bind error
	// before shutdown, including when a signal arrives during startup.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for server.Addr() == cfg.Address {
		select {
		case err := <-finished:
			stopCtx, cancelStop := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancelStop()
			return errors.Join(err, server.Shutdown(stopCtx))
		case <-ticker.C:
		}
	}
	fmt.Printf("HTTP listening on http://%s\n", server.Addr())
	select {
	case err := <-finished:
		stopCtx, cancelStop := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancelStop()
		return errors.Join(err, server.Shutdown(stopCtx))
	case <-ctx.Done():
		stopCtx, cancelStop := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancelStop()
		shutdownErr := server.Shutdown(stopCtx)
		serveErr := <-finished
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(shutdownErr, serveErr)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
