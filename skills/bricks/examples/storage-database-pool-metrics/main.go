package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/database"
	listener "github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"gorm.io/gorm"
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

func query(ctx context.Context, db *gorm.DB) error {
	var value int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
		return fmt.Errorf("query PostgreSQL: %w", err)
	}
	if value != 1 {
		return fmt.Errorf("unexpected query result: %d", value)
	}
	fmt.Println("postgres=1")
	return nil
}

func registerWork(lc fx.Lifecycle, db *gorm.DB) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return query(ctx, db)
	}})
}

func run() error {
	app := fx.New(
		database.Module,
		fx.Provide(newRegistry),
		database.PoolMetricsModule(),
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
