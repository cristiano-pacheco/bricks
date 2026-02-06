package chi

import "time"

// Option is a functional option for configuring the Server.
type Option func(*Config)

// WithHost sets the server host.
func WithHost(host string) Option {
	return func(c *Config) {
		c.Host = host
	}
}

// WithPort sets the server port.
func WithPort(port uint) Option {
	return func(c *Config) {
		c.Port = port
	}
}

// WithReadTimeout sets the read timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ReadTimeout = timeout
	}
}

// WithWriteTimeout sets the write timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.WriteTimeout = timeout
	}
}

// WithIdleTimeout sets the idle timeout.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.IdleTimeout = timeout
	}
}

// WithShutdownTimeout sets the shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

// WithHealthCheck enables or disables the health check endpoint.
func WithHealthCheck(enabled bool) Option {
	return func(c *Config) {
		c.EnableHealthCheck = enabled
	}
}

// WithHealthCheckPath sets the health check endpoint path.
func WithHealthCheckPath(path string) Option {
	return func(c *Config) {
		c.HealthCheckPath = path
	}
}

// WithCORS sets CORS configuration.
func WithCORS(cors *CORSConfig) Option {
	return func(c *Config) {
		c.CORS = cors
	}
}

// WithDefaultCORS sets a permissive CORS configuration.
func WithDefaultCORS() Option {
	return func(c *Config) {
		c.CORS = &CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           300,
		}
	}
}

// WithMetrics enables the metrics endpoint on a separate HTTP server.
func WithMetrics(enabled bool) Option {
	return func(c *Config) {
		c.EnableMetrics = enabled
	}
}

// WithMetricsPort sets the port for the metrics server.
func WithMetricsPort(port uint) Option {
	return func(c *Config) {
		c.MetricsPort = port
	}
}

// WithMetricsPath sets the path for the metrics endpoint.
func WithMetricsPath(path string) Option {
	return func(c *Config) {
		c.MetricsPath = path
	}
}
