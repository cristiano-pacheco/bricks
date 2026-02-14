package chi

import (
	"fmt"
	"time"
)

const (
	defaultPort        = 8080
	defaultMetricsPort = 9090
	defaultMaxAge      = 300
	defaultSwaggerPath = "/swagger/*"
	healthCheckPath    = "/healthz"
	metricsPath        = "/metrics"
)

// Config holds the configuration for the Chi HTTP server.
type Config struct {
	Port            uint
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MetricsPort     uint
	CORS            *CORSConfig
	Swagger         *SwaggerConfig
}

// SwaggerConfig holds configuration for the Swagger/OpenAPI documentation endpoint.
// When Enabled is false, the swagger route is not registered.
// Dir is the directory containing swagger files (e.g. doc.json from swag init).
// When Dir is empty, embedded docs from the application's docs package are used
// (the application must import its docs package, e.g. _ "myapp/docs").
type SwaggerConfig struct {
	Enabled bool   // Whether to register the swagger route
	Path    string // URL path prefix for swagger UI (e.g. /swagger), default: /swagger
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	ExposedHeaders     []string
	AllowCredentials   bool
	MaxAge             int
	OptionsPassthrough bool
	Debug              bool
}

// Default returns a Config with sensible default values.
func Default() Config {
	return Config{
		Port:            defaultPort,
		ReadTimeout:     DefaultReadTimeout * time.Second,
		WriteTimeout:    DefaultWriteTimeout * time.Second,
		IdleTimeout:     DefaultIdleTimeout * time.Second,
		ShutdownTimeout: DefaultShutdownTimeout * time.Second,
		MetricsPort:     defaultMetricsPort,
	}
}

// Validate validates the server configuration.
func (c Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidPort, c.Port)
	}
	if c.MetricsPort <= 0 || c.MetricsPort > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidMetricsPort, c.MetricsPort)
	}
	if c.Port == c.MetricsPort {
		return ErrPortsEqual
	}
	return nil
}

// WithDefaultCORS returns a new Config with permissive CORS settings.
func (c Config) WithDefaultCORS() Config {
	c.CORS = &CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           defaultMaxAge,
	}
	return c
}
