package chi

import "time"

// Config holds the configuration for the Chi HTTP server.
type Config struct {
	Host              string
	Port              uint
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	EnableHealthCheck bool
	HealthCheckPath   string
	CORS              *CORSConfig
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

func defaultConfig() Config {
	return Config{
		Host:              "localhost",
		Port:              8080,
		ReadTimeout:       DefaultReadTimeout * time.Second,
		WriteTimeout:      DefaultWriteTimeout * time.Second,
		IdleTimeout:       DefaultIdleTimeout * time.Second,
		ShutdownTimeout:   DefaultShutdownTimeout * time.Second,
		EnableHealthCheck: true,
		HealthCheckPath:   "/healthz",
	}
}
