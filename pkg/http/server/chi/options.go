package chi

import "time"

const defaultCORSMaxAge = 300

// Option is a functional option for configuring the Server.
type Option func(*Config)

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

// WithMetricsPort sets the port for the metrics server.
func WithMetricsPort(port uint) Option {
	return func(c *Config) {
		c.MetricsPort = port
	}
}

// WithSwagger enables swagger and sets its configuration.
func WithSwagger(enabled bool, swaggerPath, dir string) Option {
	return func(c *Config) {
		c.Swagger = &SwaggerConfig{
			Enabled: enabled,
			Path:    swaggerPath,
		}
	}
}
