package ucdecorator

// Config holds the configuration for the use case decorator chain.
type Config struct {
	// Enabled controls whether any decorators are applied.
	// When false, Wrap returns the handler unchanged regardless of other flags.
	Enabled bool `config:"enabled"`

	// Logging controls whether the logging decorator is applied.
	Logging bool `config:"logging"`

	// Metrics controls whether the metrics decorator is applied.
	Metrics bool `config:"metrics"`

	// Tracing controls whether the tracing decorator is applied.
	Tracing bool `config:"tracing"`

	// Translation controls whether the error translation decorator is applied.
	Translation bool `config:"translation"`
}

// DefaultConfig returns a configuration with all decorators enabled.
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		Logging:     true,
		Metrics:     true,
		Tracing:     true,
		Translation: true,
	}
}
