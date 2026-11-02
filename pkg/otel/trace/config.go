package trace

import (
	"time"

	oteltrace "github.com/cristiano-pacheco/go-otel/trace"
)

type TracerConfig struct {
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
	SampleRate      float64       `config:"sample_rate"`
	SampleRateSet   bool          `config:"sample_rate_set"`
	ExporterType    ExporterType  `config:"exporter_type"`
}

// Validate checks the tracing configuration.
func (c *TracerConfig) Validate() error {
	cfg := c.otelConfig()
	return cfg.Validate()
}

func (c *TracerConfig) otelConfig() oteltrace.TracerConfig {
	return oteltrace.TracerConfig{
		AppName: c.AppName, AppVersion: c.AppVersion, TracerVendor: c.TracerVendor,
		TraceURL: c.TraceURL, TraceEnabled: c.TraceEnabled, BatchTimeout: c.BatchTimeout,
		MaxBatchSize: c.MaxBatchSize, MaxQueueSize: c.MaxQueueSize, ExportTimeout: c.ExportTimeout,
		ShutdownTimeout: c.ShutdownTimeout, Insecure: c.Insecure, SampleRate: c.SampleRate,
		SampleRateSet: c.SampleRateSet, ExporterType: c.ExporterType,
	}
}
