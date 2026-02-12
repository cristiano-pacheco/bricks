package trace

import (
	"time"
)

const (
	defaultBatchTimeout = 5 * time.Second
	defaultSampleRate   = 0.01
)

type TracerConfig struct {
	AppName      string        `config:"app_name"`       // Application name
	AppVersion   string        `config:"app_version"`    // Application version
	TracerVendor string        `config:"tracer_vendor"`  // Tracer vendor name (e.g., jaeger, tempo, datadog)
	TraceURL     string        `config:"trace_url"`      // OTLP endpoint URL (without http/https prefix)
	TraceEnabled bool          `config:"trace_enabled"`  // Enable/disable tracing
	BatchTimeout time.Duration `config:"batch_timeout"`  // Maximum time to wait before sending a batch
	MaxBatchSize int           `config:"max_batch_size"` // Maximum number of spans in a batch
	Insecure     bool          `config:"insecure"`       // Use insecure connection (no TLS)
	SampleRate   float64       `config:"sample_rate"`    // 0.0 to 1.0
	ExporterType ExporterType  `config:"exporter_type"`  // GRPC or HTTP, default GRPC
}

// Validate checks if the configuration is valid
func (c *TracerConfig) Validate() error {
	if c.AppName == "" {
		return ErrAppNameRequired
	}
	if c.TraceEnabled && c.TraceURL == "" {
		return ErrTraceURLRequired
	}
	if c.SampleRate < 0.0 || c.SampleRate > 1.0 {
		return ErrInvalidSampleRate
	}
	return nil
}

// setDefaults sets default values for optional configuration fields
func (c *TracerConfig) setDefaults() {
	if c.BatchTimeout == 0 {
		c.BatchTimeout = defaultBatchTimeout
	}
	if c.MaxBatchSize == 0 {
		c.MaxBatchSize = 512
	}
	if c.SampleRate == 0.0 {
		c.SampleRate = defaultSampleRate
	}
	if c.ExporterType.IsZero() {
		exporterType, err := NewExporterType(ExporterTypeGRPC)
		if err == nil {
			c.ExporterType = exporterType
		}
	}
}
