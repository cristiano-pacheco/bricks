package chi

import "time"

const defaultCORSMaxAge = 300

// Option is a functional option for configuring the Server.
type Option func(*Config)

// WithAddress sets the complete TCP listen address for the server.
// When it is empty, the server uses Port with the default listen host.
func WithAddress(address string) Option {
	return func(c *Config) {
		c.Address = address
	}
}

// WithPort sets the server port and clears a previously configured address.
func WithPort(port uint) Option {
	return func(c *Config) {
		c.Address = ""
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
			MaxAge:           defaultCORSMaxAge,
		}
	}
}

// WithMetrics enables or disables the metrics listener.
func WithMetrics(enabled bool) Option {
	return func(c *Config) {
		c.EnableMetrics = enabled
	}
}

// WithMetricsAddress sets the complete TCP listen address for the metrics server.
// When it is empty, the metrics server uses MetricsPort with the default listen host.
func WithMetricsAddress(address string) Option {
	return func(c *Config) {
		c.MetricsAddress = address
	}
}

// WithMetricsPort sets the metrics server port and clears a previously configured address.
func WithMetricsPort(port uint) Option {
	return func(c *Config) {
		c.MetricsAddress = ""
		c.MetricsPort = port
	}
}

// WithRequestLogging enables or disables request logging.
func WithRequestLogging(enabled bool) Option {
	return func(c *Config) {
		c.EnableRequestLogging = enabled
	}
}

// WithRouteLogging enables or disables route logging.
func WithRouteLogging(enabled bool) Option {
	return func(c *Config) {
		c.EnableRouteLogging = enabled
	}
}

// WithSwagger enables swagger and sets its configuration.
func WithSwagger(enabled bool, swaggerPath string) Option {
	return func(c *Config) {
		c.Swagger = &SwaggerConfig{
			Enabled: enabled,
			Path:    swaggerPath,
		}
	}
}
