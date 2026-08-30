package chi

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenHost  = "0.0.0.0"
	defaultPort        = 8080
	defaultMetricsPort = 9090
	defaultMaxAge      = 300
	defaultSwaggerPath = "/swagger/*"
	healthCheckPath    = "/healthz"
	metricsPath        = "/metrics"
	maxPort            = 65535
	portBitSize        = 16
)

// Config holds the configuration for the Chi HTTP server.
type Config struct {
	Address              string         `config:"address"`
	Port                 uint           `config:"port"`
	ReadTimeout          time.Duration  `config:"readtimeout"`
	WriteTimeout         time.Duration  `config:"writetimeout"`
	IdleTimeout          time.Duration  `config:"idletimeout"`
	ShutdownTimeout      time.Duration  `config:"shutdowntimeout"`
	EnableMetrics        bool           `config:"enablemetrics"`
	MetricsAddress       string         `config:"metricsaddress"`
	MetricsPort          uint           `config:"metricsport"`
	EnableRequestLogging bool           `config:"enablerequestlogging"`
	EnableRouteLogging   bool           `config:"enableroutelogging"`
	CORS                 *CORSConfig    `config:"cors"`
	Swagger              *SwaggerConfig `config:"swagger"`
}

// SwaggerConfig holds configuration for the Swagger/OpenAPI documentation endpoint.
// When Enabled is false, the swagger route is not registered.
// Dir is the directory containing swagger files (e.g. doc.json from swag init).
// When Dir is empty, embedded docs from the application's docs package are used
// (the application must import its docs package, e.g. _ "myapp/docs").
type SwaggerConfig struct {
	Enabled bool   `config:"enabled"` // Whether to register the swagger route.
	Path    string `config:"path"`    // URL path prefix for swagger UI.
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins     []string `config:"allowedorigins"`
	AllowedMethods     []string `config:"allowedmethods"`
	AllowedHeaders     []string `config:"allowedheaders"`
	ExposedHeaders     []string `config:"exposedheaders"`
	AllowCredentials   bool     `config:"allowcredentials"`
	MaxAge             int      `config:"maxage"`
	OptionsPassthrough bool     `config:"optionspassthrough"`
	Debug              bool     `config:"debug"`
}

// Default returns a Config with sensible default values.
func Default() Config {
	return Config{
		Port:                 defaultPort,
		ReadTimeout:          DefaultReadTimeout * time.Second,
		WriteTimeout:         DefaultWriteTimeout * time.Second,
		IdleTimeout:          DefaultIdleTimeout * time.Second,
		ShutdownTimeout:      DefaultShutdownTimeout * time.Second,
		EnableMetrics:        true,
		MetricsPort:          defaultMetricsPort,
		EnableRequestLogging: true,
		EnableRouteLogging:   true,
	}
}

// Validate validates the server configuration.
func (c Config) Validate() error {
	mainPort, err := validateListenAddress(c.listenAddress(), ErrInvalidAddress, ErrInvalidPort)
	if err != nil {
		return err
	}
	if !c.EnableMetrics {
		return nil
	}

	metricsPort, err := validateListenAddress(c.metricsListenAddress(), ErrInvalidMetricsAddress, ErrInvalidMetricsPort)
	if err != nil {
		return err
	}
	if mainPort != 0 && mainPort == metricsPort {
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

func (c Config) listenAddress() string {
	if address := strings.TrimSpace(c.Address); address != "" {
		return address
	}
	return net.JoinHostPort(defaultListenHost, strconv.FormatUint(uint64(c.Port), 10))
}

func (c Config) metricsListenAddress() string {
	if address := strings.TrimSpace(c.MetricsAddress); address != "" {
		return address
	}
	return net.JoinHostPort(defaultListenHost, strconv.FormatUint(uint64(c.MetricsPort), 10))
}

func validateListenAddress(address string, invalidAddress, invalidPort error) (uint, error) {
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", invalidAddress, address)
	}

	port, err := strconv.ParseUint(portText, 10, portBitSize)
	if err != nil || port > maxPort {
		return 0, fmt.Errorf("%w: %s", invalidPort, portText)
	}
	return uint(port), nil
}
