package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	"go.uber.org/fx"
)

func registerOrder(lifecycle fx.Lifecycle, measurements metrics.UseCaseMetrics) {
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		started := time.Now()
		err := createOrder(ctx)
		measurements.ObserveDuration("create_order", time.Since(started))
		if err != nil {
			measurements.IncError("create_order")
			return err
		}
		measurements.IncSuccess("create_order")
		return nil
	}})
}

func createOrder(ctx context.Context) error {
	return ctx.Err()
}

func run() error {
	app := fx.New(
		metrics.Module,
		chi.Module,
		fx.Invoke(registerOrder),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	err := app.Start(startCtx)
	cancelStart()
	if err == nil {
		signal := <-app.Wait()
		if signal.ExitCode != 0 {
			err = fmt.Errorf("application requested exit code %d", signal.ExitCode)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
