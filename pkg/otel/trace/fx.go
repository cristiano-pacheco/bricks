package trace

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

// Module loads tracing configuration and manages the tracer lifecycle.
var Module = fx.Module("otel.trace", config.Provide[yamlConfig]("app.open-telemetry"),
	fx.Provide(newTraceLifecycle), fx.Invoke(func(*traceLifecycle) {}))

type traceLifecycle struct{}

func newTraceLifecycle(lc fx.Lifecycle, value config.Config[yamlConfig]) (*traceLifecycle, error) {
	if err := initializeYAML(lc, value); err != nil {
		return nil, err
	}

	return &traceLifecycle{}, nil
}

type yamlConfig struct {
	AppName         string        `config:"app_name"`
	AppVersion      string        `config:"app_version"`
	TracerVendor    string        `config:"tracer_vendor"`
	TraceURL        string        `config:"trace_url"`
	TraceEnabled    bool          `config:"trace_enabled"`
	BatchTimeout    time.Duration `config:"batch_timeout"`
	MaxBatchSize    int           `config:"max_batch_size"`
	MaxQueueSize    int           `config:"max_queue_size"`
	ExportTimeout   time.Duration `config:"export_timeout"`
	ShutdownTimeout time.Duration `config:"shutdown_timeout"`
	Insecure        bool          `config:"insecure"`
	SampleRate      *float64      `config:"sample_rate"`
	ExporterType    string        `config:"exporter_type"`
}

func initializeYAML(lc fx.Lifecycle, value config.Config[yamlConfig]) error {
	c := value.Get()

	cfg := TracerConfig{
		AppName: c.AppName, AppVersion: c.AppVersion, TracerVendor: c.TracerVendor,
		TraceURL: c.TraceURL, TraceEnabled: c.TraceEnabled, BatchTimeout: c.BatchTimeout,
		MaxBatchSize: c.MaxBatchSize, MaxQueueSize: c.MaxQueueSize, ExportTimeout: c.ExportTimeout,
		ShutdownTimeout: c.ShutdownTimeout, Insecure: c.Insecure,
	}
	if c.SampleRate != nil {
		cfg.SampleRate, cfg.SampleRateSet = *c.SampleRate, true
	}

	if c.ExporterType != "" {
		var err error

		cfg.ExporterType, err = NewExporterType(c.ExporterType)
		if err != nil {
			return err
		}
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	appendLifecycle(lc, cfg)

	return nil
}

// InitializeWithLifecycle registers initialization and shutdown with Fx.
func InitializeWithLifecycle(lc fx.Lifecycle, cfg config.Config[TracerConfig]) error {
	c := cfg.Get()
	if err := c.Validate(); err != nil {
		return err
	}

	appendLifecycle(lc, c)

	return nil
}

func appendLifecycle(lc fx.Lifecycle, cfg TracerConfig) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return Initialize(cfg)
		},
		OnStop: Shutdown,
	})
}
