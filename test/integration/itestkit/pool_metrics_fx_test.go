//go:build integration

package itestkit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/database"
	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type poolQueryService struct {
	db *gorm.DB
}

func (s *poolQueryService) Execute(ctx context.Context) error {
	return s.db.WithContext(ctx).Exec("SELECT 1").Error
}

func TestPoolMetrics_PublicService(t *testing.T) {
	kit := itestkit.New(itestkit.DefaultConfig())
	require.NoError(t, kit.StartPostgres())
	t.Cleanup(kit.StopPostgres)

	db := kit.DB()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(3)
	registry := prometheus.NewRegistry()
	service := &poolQueryService{db: db}
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "base.yaml")
	t.Setenv("APP_CONFIG_DIR", configDir)
	t.Setenv("APP_ENV", "local")
	require.NoError(t, os.WriteFile(configFile, []byte("app:\n  name: test\n"), 0600))
	newApp := func() *fx.App {
		return fx.New(
			fx.Supply(db),
			fx.Supply(fx.Annotate(registry, fx.As(new(prometheus.Registerer)))),
			database.PoolMetricsModule(),
			fx.NopLogger,
		)
	}

	for _, yaml := range []string{
		"app:\n  name: test\n",
		"app:\n  database: {}\n",
		"app:\n  database:\n    metrics: {}\n",
		"app:\n  database:\n    tracing:\n      enabled: true\n",
	} {
		require.NoError(t, os.WriteFile(configFile, []byte(yaml), 0600))
		disabled := newApp()
		require.NoError(t, disabled.Err())
		require.NoError(t, disabled.Start(context.Background()))
		require.NoError(t, service.Execute(context.Background()))
		families, gatherErr := registry.Gather()
		require.NoError(t, gatherErr)
		require.Empty(t, families)
		require.NoError(t, disabled.Stop(context.Background()))
	}

	dbConstructed := false
	registryConstructed := false
	withoutDependencies := fx.New(database.PoolMetricsModule(), fx.NopLogger, fx.Provide(
		func() *gorm.DB {
			dbConstructed = true
			return db
		},
		func() prometheus.Registerer {
			registryConstructed = true
			return registry
		},
	))
	require.NoError(t, withoutDependencies.Err())
	require.NoError(t, withoutDependencies.Start(context.Background()))
	require.NoError(t, withoutDependencies.Stop(context.Background()))
	require.False(t, dbConstructed)
	require.False(t, registryConstructed)

	require.NoError(t, os.WriteFile(configFile, []byte("app:\n  database:\n    metrics:\n      enabled: invalid\n"), 0600))
	invalid := fx.New(database.PoolMetricsModule(), fx.NopLogger)
	require.Error(t, invalid.Err())

	require.NoError(t, os.WriteFile(configFile, []byte("app:\n  database:\n    metrics:\n      enabled: true\n"), 0600))
	missingDependencies := fx.New(database.PoolMetricsModule(), fx.NopLogger)
	require.Error(t, missingDependencies.Err())
	app := newApp()
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.Background()))
	require.NoError(t, service.Execute(context.Background()))

	families, err := registry.Gather()
	require.NoError(t, err)
	values := make(map[string]float64)
	for _, family := range families {
		require.Len(t, family.Metric, 1)
		metric := family.Metric[0]
		require.Len(t, metric.Label, 1)
		require.Equal(t, "db_name", metric.Label[0].GetName())
		require.Equal(t, "postgres", metric.Label[0].GetValue())
		if metric.Gauge != nil {
			values[family.GetName()] = metric.Gauge.GetValue()
		} else if metric.Counter != nil {
			values[family.GetName()] = metric.Counter.GetValue()
		}
	}
	stats := sqlDB.Stats()
	require.Equal(t, float64(stats.MaxOpenConnections), values["go_sql_max_open_connections"])
	require.Equal(t, float64(stats.OpenConnections), values["go_sql_open_connections"])
	require.Equal(t, float64(stats.InUse), values["go_sql_in_use_connections"])
	require.Equal(t, float64(stats.Idle), values["go_sql_idle_connections"])
	require.Equal(t, float64(stats.WaitCount), values["go_sql_wait_count_total"])
	require.Equal(t, stats.WaitDuration.Seconds(), values["go_sql_wait_duration_seconds_total"])

	duplicate := newApp()
	require.NoError(t, duplicate.Err())
	require.Error(t, duplicate.Start(context.Background()))
	require.NoError(t, service.Execute(context.Background()))
	require.NoError(t, app.Stop(context.Background()))
	families, err = registry.Gather()
	require.NoError(t, err)
	require.Empty(t, families)
	require.NoError(t, service.Execute(context.Background()))

	config := "app:\n  database:\n    metrics:\n      enabled: true\n    tracing:\n      enabled: true\n"
	require.NoError(t, os.WriteFile(configFile, []byte(config), 0600))
	second := newApp()
	require.NoError(t, second.Err())
	require.NoError(t, second.Start(context.Background()))
	require.NoError(t, service.Execute(context.Background()))
	require.NoError(t, second.Stop(context.Background()))
}
