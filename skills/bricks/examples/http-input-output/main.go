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
	"github.com/cristiano-pacheco/bricks/pkg/http/request"
	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	pathrequest "github.com/cristiano-pacheco/bricks/pkg/http/server/chi/request"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/validator"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

type UserInput struct {
	Name string `json:"name" validate:"required"`
}

type CreateUser struct {
	validate validator.Validator
}

func (uc *CreateUser) Execute(input UserInput) (UserInput, error) {
	if err := uc.validate.Validate(input); err != nil {
		return UserInput{}, err
	}
	return input, nil
}

type endpoint func(http.ResponseWriter, *http.Request) error

func adapt(handler endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			// A response may already have started. Do not write a second response.
			slog.Error("write HTTP response", "error", err)
		}
	}
}

func routes(v validator.Validator, h response.ErrorHandler) http.Handler {
	r := chi.NewRouter()
	uc := &CreateUser{validate: v}
	decode := func(limit int64, validate bool) http.HandlerFunc {
		return adapt(func(w http.ResponseWriter, r *http.Request) error {
			var input UserInput
			if err := request.ReadJSONWithMaxSize(w, r, &input, limit); err != nil {
				h.Error(w, err)
				return nil
			}
			if validate {
				output, err := uc.Execute(input)
				if err != nil {
					h.Error(w, err)
					return nil
				}
				return response.JSON(w, http.StatusCreated, output, nil)
			}
			return response.JSON(w, http.StatusCreated, input, nil)
		})
	}
	r.Post("/decode", decode(request.DefaultMaxBodySize, false))
	r.Post("/small", decode(32, false))
	r.Post("/users", decode(request.DefaultMaxBodySize, true))
	r.Get("/users/{id}", adapt(func(w http.ResponseWriter, r *http.Request) error {
		return response.JSON(w, http.StatusOK, map[string]string{"id": pathrequest.Param(r, "id")}, nil)
	}))
	r.Get("/enveloped", adapt(func(w http.ResponseWriter, _ *http.Request) error {
		return response.JSON(w, http.StatusOK, map[string]string{"name": "Ada"}, http.Header{"X-Example": {"bricks"}})
	}))
	r.Get("/raw", adapt(func(w http.ResponseWriter, _ *http.Request) error {
		return response.JSONRaw(w, http.StatusOK, map[string]string{"name": "Ada"}, nil)
	}))
	r.Delete("/users/{id}", func(w http.ResponseWriter, _ *http.Request) { response.NoContent(w) })
	r.Get("/structured-error", func(w http.ResponseWriter, _ *http.Request) {
		h.Error(w, errs.New("USER_NOT_FOUND", "user not found", http.StatusNotFound, nil))
	})
	r.Get("/unknown-error", func(w http.ResponseWriter, _ *http.Request) {
		h.Error(w, errors.New("private database credentials must not reach the client"))
	})
	return r
}

func provideLogger(lc fx.Lifecycle) (logger.Logger, error) {
	// Disable log sinks for this example. The response handler still receives its
	// required logger interface, while transport write failures use slog.
	log, err := logger.NewWithOptions(logger.WithOutputPaths(), logger.WithErrorOutputPaths())
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return log.Sync() }})
	return log, nil
}

func serve(lc fx.Lifecycle, handler http.Handler, shutdown fx.Shutdowner) {
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}
			go func() {
				err := server.Serve(listener)
				if errors.Is(err, http.ErrServerClosed) {
					err = nil
				}
				done <- err
				if err != nil {
					if stopErr := shutdown.Shutdown(); stopErr != nil {
						slog.Error("request application shutdown", "error", stopErr)
					}
				}
			}()
			fmt.Println("HTTP listening on http://127.0.0.1:8080")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := server.Shutdown(ctx)
			if err != nil {
				err = errors.Join(err, server.Close())
			}
			return errors.Join(err, <-done)
		},
	})
}

func run() (err error) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	app := fx.New(validator.Module, response.Module, fx.Provide(provideLogger, routes), fx.Invoke(serve))
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(ctx, 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
	case <-app.Done():
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	return app.Stop(stopCtx)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
