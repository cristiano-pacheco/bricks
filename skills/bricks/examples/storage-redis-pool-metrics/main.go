package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

type registryValues struct {
	fx.Out
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
}

func newRegistry() registryValues {
	registry := prometheus.NewRegistry()
	return registryValues{Registerer: registry, Gatherer: registry}
}

func underlying(client *bricksredis.Client) bricksredis.UniversalClient {
	return client.UniversalClient()
}

func registerWork(lc fx.Lifecycle, client bricksredis.UniversalClient) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		value, err := client.Ping(ctx).Result()
		if err != nil {
			return fmt.Errorf("ping Redis: %w", err)
		}
		fmt.Println("redis=" + value)
		return nil
	}})
}

func run() error {
	app := fx.New(
		bricksredis.ClientModule,
		fx.Provide(underlying, newRegistry),
		bricksredis.PoolMetricsModule(),
		listener.Module,
		fx.Invoke(registerWork),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()
	if startErr == nil {
		signal := <-app.Wait()
		if signal.ExitCode != 0 {
			startErr = fmt.Errorf("application requested exit code %d", signal.ExitCode)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
