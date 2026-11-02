package trace

import (
	"context"
	"errors"
	"fmt"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// GORMModule manages tracing and instruments the supplied GORM client when
// app.database.tracing.enabled is true. Declare it instead of Module.
func GORMModule() fx.Option {
	cfg, err := config.New[gormYAMLConfig]()
	if err != nil {
		return fx.Options(Module, fx.Error(fmt.Errorf("load GORM tracing configuration: %w", err)))
	}

	if !cfg.Get().App.Database.Tracing.Enabled {
		return Module
	}

	return fx.Module("otel.trace.gorm",
		Module,
		fx.Provide(newGORMInstrumentation),
		fx.Invoke(func(*gormInstrumentation) {}),
		fx.Invoke(instrumentGORM),
	)
}

type gormYAMLConfig struct {
	App struct {
		Database struct {
			Tracing struct {
				Enabled bool `config:"enabled"`
			} `config:"tracing"`
		} `config:"database"`
	} `config:"app"`
}

type gormParams struct {
	fx.In

	Instrumentation *gormInstrumentation
	DB              *gorm.DB
}

func newGORMInstrumentation(lifecycle fx.Lifecycle) *gormInstrumentation {
	instrumentation := &gormInstrumentation{}
	lifecycle.Append(fx.Hook{OnStart: instrumentation.start, OnStop: instrumentation.stop})

	return instrumentation
}

func instrumentGORM(params gormParams) error {
	if params.DB == nil || params.DB.Config == nil || params.DB.Config.Plugins == nil ||
		params.DB.Dialector == nil || params.DB.Callback() == nil {
		return errors.New("GORM tracing requires an initialized GORM client")
	}

	params.Instrumentation.db = params.DB

	return nil
}

type gormCallbackRegistry interface {
	Get(string) func(*gorm.DB)
	Remove(string) error
}

type gormCallback struct {
	registry gormCallbackRegistry
	name     string
}

type gormInstrumentation struct {
	db     *gorm.DB
	plugin gorm.Plugin
}

// callbacks lists the callback names registered by the pinned plugin version.
func (i *gormInstrumentation) callbacks() []gormCallback {
	callbacks := i.db.Callback()

	return []gormCallback{
		{callbacks.Create(), "otel:before:create"}, {callbacks.Create(), "otel:after:create"},
		{callbacks.Query(), "otel:before:select"}, {callbacks.Query(), "otel:after:select"},
		{callbacks.Delete(), "otel:before:delete"}, {callbacks.Delete(), "otel:after:delete"},
		{callbacks.Update(), "otel:before:update"}, {callbacks.Update(), "otel:after:update"},
		{callbacks.Row(), "otel:before:row"}, {callbacks.Row(), "otel:after:row"},
		{callbacks.Raw(), "otel:before:raw"}, {callbacks.Raw(), "otel:after:raw"},
	}
}

func (i *gormInstrumentation) start(_ context.Context) error {
	plugin := tracing.NewPlugin(
		tracing.WithTracerProvider(otel.GetTracerProvider()),
		tracing.WithoutMetrics(),
		tracing.WithoutQueryVariables(),
		tracing.WithQueryFormatter(func(string) string {
			return ""
		}),
	)

	if _, exists := i.db.Plugins[plugin.Name()]; exists {
		return fmt.Errorf("install GORM tracing: %w", gorm.ErrRegistered)
	}

	for _, callback := range i.callbacks() {
		if callback.registry.Get(callback.name) != nil {
			return fmt.Errorf("install GORM tracing: callback %q already exists", callback.name)
		}
	}

	if err := i.db.Use(plugin); err != nil {
		return errors.Join(fmt.Errorf("install GORM tracing: %w", err), i.removeCallbacks())
	}

	i.plugin = plugin

	return nil
}

func (i *gormInstrumentation) stop(_ context.Context) error {
	if i.plugin == nil {
		return nil
	}

	err := i.removeCallbacks()
	if i.db.Plugins[i.plugin.Name()] == i.plugin {
		delete(i.db.Plugins, i.plugin.Name())
	}

	i.plugin = nil

	return err
}

func (i *gormInstrumentation) removeCallbacks() error {
	var result error

	for _, callback := range i.callbacks() {
		if callback.registry.Get(callback.name) == nil {
			continue
		}

		if err := callback.registry.Remove(callback.name); err != nil {
			result = errors.Join(result, fmt.Errorf("remove GORM tracing callback %q: %w", callback.name, err))
		}
	}

	return result
}
