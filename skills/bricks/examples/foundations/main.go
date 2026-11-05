package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"go.uber.org/fx"
)

type ServiceConfig struct {
	Name string `config:"name"`
	Port int    `config:"port"`
}

func registerService(lc fx.Lifecycle, cfg config.Config[ServiceConfig], log logger.Logger) error {
	service := cfg.Get()
	if service.Name == "" || service.Port < 1 || service.Port > 65535 {
		return errors.New("service requires a name and a port between 1 and 65535")
	}

	child := log.Named(service.Name).With(logger.Int("port", service.Port))
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			child.Info("service started")
			problem := errs.New("EXAMPLE_REJECTION", "Example request rejected", http.StatusUnprocessableEntity,
				[]errs.Detail{{Field: "email", Message: "must be a valid address"}})
			problem.OriginalError = errors.New("example diagnostic cause")
			child.Warn("example application error", logger.String("code", problem.Code), logger.Error(problem))
			return nil
		},
		OnStop: func(context.Context) error {
			child.Info("service stopped")
			return nil
		},
	})
	return nil
}

func run() error {
	app := fx.New(
		logger.Module,
		config.Provide[ServiceConfig]("app.service"),
		fx.Invoke(registerService),
	)
	if err := app.Err(); err != nil {
		return err
	}

	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err := app.Start(startCtx)
	cancelStart()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
