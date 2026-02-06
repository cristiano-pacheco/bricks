package logger

// Option is a functional option for configuring the logger
type Option func(*Config)

// WithLevel sets the log level
func WithLevel(level string) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithEncoding sets the encoding format (json or console)
func WithEncoding(encoding string) Option {
	return func(c *Config) {
		c.Encoding = encoding
	}
}

// WithDevelopment enables or disables development mode
func WithDevelopment(dev bool) Option {
	return func(c *Config) {
		c.Development = dev
	}
}

// WithCaller enables or disables caller information
func WithCaller(enabled bool) Option {
	return func(c *Config) {
		c.DisableCaller = !enabled
	}
}

// WithStacktrace enables or disables stacktrace
func WithStacktrace(enabled bool) Option {
	return func(c *Config) {
		c.DisableStacktrace = !enabled
	}
}

// WithOutputPaths sets the output paths
func WithOutputPaths(paths ...string) Option {
	return func(c *Config) {
		c.OutputPaths = paths
	}
}

// WithErrorOutputPaths sets the error output paths
func WithErrorOutputPaths(paths ...string) Option {
	return func(c *Config) {
		c.ErrorOutputPaths = paths
	}
}

// WithInitialFields adds initial fields to every log entry
func WithInitialFields(fields map[string]interface{}) Option {
	return func(c *Config) {
		if c.InitialFields == nil {
			c.InitialFields = make(map[string]interface{})
		}
		for k, v := range fields {
			c.InitialFields[k] = v
		}
	}
}

// WithField adds a single initial field
func WithField(key string, value interface{}) Option {
	return func(c *Config) {
		if c.InitialFields == nil {
			c.InitialFields = make(map[string]interface{})
		}
		c.InitialFields[key] = value
	}
}

// WithSampling enables log sampling
func WithSampling(initial, thereafter int) Option {
	return func(c *Config) {
		c.SamplingConfig = &SamplingConfig{
			Initial:    initial,
			Thereafter: thereafter,
		}
	}
}

// NewWithOptions creates a logger with functional options
func NewWithOptions(opts ...Option) (*ZapLogger, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return New(config)
}

// MustNewWithOptions creates a logger with options or panics
func MustNewWithOptions(opts ...Option) *ZapLogger {
	logger, err := NewWithOptions(opts...)
	if err != nil {
		panic(err)
	}
	return logger
}
