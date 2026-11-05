package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"

	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
)

func newGatherer() (prometheus.Gatherer, error) {
	registry := prometheus.NewRegistry()

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total", Help: "Orders created by this application.",
	})
	if err := registry.Register(counter); err != nil {
		return nil, err
	}

	counter.Inc()

	return registry, nil
}

func run() error {
	app := fx.New(listener.Module, fx.Provide(newGatherer))
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
