package redis

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

// PoolMetricsModule registers pool metrics for an application-owned Redis client.
func PoolMetricsModule() fx.Option {
	cfg, err := config.New[poolMetricsConfig]()
	if err != nil {
		return fx.Error(fmt.Errorf("load Redis pool metrics configuration: %w", err))
	}

	if !cfg.Get().App.Redis.Metrics.Enabled {
		return fx.Options()
	}

	return fx.Module("redis.pool-metrics", fx.Invoke(registerPoolMetrics))
}

type poolMetricsConfig struct {
	App struct {
		Redis struct {
			Metrics struct {
				Enabled bool `config:"enabled"`
			} `config:"metrics"`
		} `config:"redis"`
	} `config:"app"`
}
type poolMetricsParams struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Client     UniversalClient
	Registerer prometheus.Registerer
}

func registerPoolMetrics(params poolMetricsParams) error {
	if params.Client == nil ||
		reflect.ValueOf(params.Client).Kind() == reflect.Pointer && reflect.ValueOf(params.Client).IsNil() ||
		params.Registerer == nil {
		return errors.New("redis pool metrics require a client and a Prometheus registerer")
	}

	collector := redisprometheus.NewCollector("redis", "", params.Client)
	params.Lifecycle.Append(fx.Hook{OnStart: func(context.Context) error {
		if err := params.Registerer.Register(collector); err != nil {
			return fmt.Errorf("register Redis pool metrics: %w", err)
		}

		return nil
	}, OnStop: func(context.Context) error { params.Registerer.Unregister(collector); return nil }})

	return nil
}
