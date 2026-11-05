package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

// PoolMetricsModule registers PostgreSQL pool metrics when app.database.metrics.enabled is true.
// The application supplies a *gorm.DB and a prometheus.Registerer and retains ownership of both.
func PoolMetricsModule() fx.Option {
	cfg, err := config.New[poolMetricsConfig]()
	if err != nil {
		return fx.Error(fmt.Errorf("load database pool metrics configuration: %w", err))
	}

	if !cfg.Get().App.Database.Metrics.Enabled {
		return fx.Options()
	}

	return fx.Module("database.pool-metrics", fx.Invoke(registerPoolMetrics))
}

type poolMetricsConfig struct {
	App struct {
		Database struct {
			Metrics struct {
				Enabled bool `config:"enabled"`
			} `config:"metrics"`
		} `config:"database"`
	} `config:"app"`
}

type poolMetricsParams struct {
	fx.In

	Lifecycle  fx.Lifecycle
	DB         *gorm.DB
	Registerer prometheus.Registerer
}

func registerPoolMetrics(params poolMetricsParams) error {
	db, registerer := params.DB, params.Registerer
	if db == nil || db.Config == nil || registerer == nil {
		return errors.New("database pool metrics require a GORM client and a Prometheus registerer")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database pool metrics: get underlying sql.DB: %w", err)
	}

	collector := collectors.NewDBStatsCollector(sqlDB, "postgres")

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if registerErr := registerer.Register(collector); registerErr != nil {
				return fmt.Errorf("register database pool metrics: %w", registerErr)
			}

			return nil
		},
		OnStop: func(context.Context) error {
			registerer.Unregister(collector)
			return nil
		},
	})

	return nil
}
