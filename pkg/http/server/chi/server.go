package chi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DefaultReadTimeout     = 15
	DefaultWriteTimeout    = 15
	DefaultIdleTimeout     = 60
	DefaultShutdownTimeout = 10
)

// Server wraps an HTTP server with Chi router.
type Server struct {
	server        *http.Server
	router        *chi.Mux
	metricsServer *http.Server
	config        Config
}

// New creates a new HTTP server with Chi router and websocket upgrader.
func New(opts ...Option) (*Server, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	router := chi.NewRouter()

	// Default middleware stack
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// CORS middleware if configured
	if cfg.CORS != nil {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:     cfg.CORS.AllowedOrigins,
			AllowedMethods:     cfg.CORS.AllowedMethods,
			AllowedHeaders:     cfg.CORS.AllowedHeaders,
			ExposedHeaders:     cfg.CORS.ExposedHeaders,
			AllowCredentials:   cfg.CORS.AllowCredentials,
			MaxAge:             cfg.CORS.MaxAge,
			OptionsPassthrough: cfg.CORS.OptionsPassthrough,
			Debug:              cfg.CORS.Debug,
		}))
	}

	// Add health check endpoint if enabled
	if cfg.EnableHealthCheck {
		router.Get(cfg.HealthCheckPath, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Create metrics server if enabled
	var metricsServer *http.Server
	if cfg.EnableMetrics {
		metricsRouter := chi.NewRouter()
		metricsRouter.Handle(cfg.MetricsPath, promhttp.Handler())

		metricsServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.MetricsPort),
			Handler:      metricsRouter,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		}
	}

	return &Server{
		server:        srv,
		router:        router,
		metricsServer: metricsServer,
		config:        cfg,
	}, nil
}

// Router returns the Chi router for registering routes.
func (s *Server) Router() *chi.Mux {
	return s.router
}

// Start begins listening and serving HTTP requests.
// If metrics are enabled, starts the metrics server on a separate port.
func (s *Server) Start() error {
	// Start metrics server if enabled
	if s.metricsServer != nil {
		go func() {
			if err := s.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				// Log error but don't stop the main server
			}
		}()
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
// Also shuts down the metrics server if enabled.
func (s *Server) Shutdown(ctx context.Context) error {
	// Shutdown metrics server if running
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			// Continue to shutdown main server even if metrics server fails
		}
	}

	return s.server.Shutdown(ctx)
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.server.Addr
}

// MetricsAddr returns the metrics server address if enabled.
func (s *Server) MetricsAddr() string {
	if s.metricsServer != nil {
		return s.metricsServer.Addr
	}
	return ""
}

func validateConfig(cfg Config) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("host cannot be empty")
	}
	if cfg.EnableHealthCheck && strings.TrimSpace(cfg.HealthCheckPath) == "" {
		return errors.New("health check path cannot be empty when health check is enabled")
	}
	if cfg.EnableMetrics {
		if cfg.MetricsPort <= 0 || cfg.MetricsPort > 65535 {
			return fmt.Errorf("invalid metrics port: %d", cfg.MetricsPort)
		}
		if cfg.Port == cfg.MetricsPort {
			return errors.New("metrics port must be different from main server port")
		}
		if strings.TrimSpace(cfg.MetricsPath) == "" {
			return errors.New("metrics path cannot be empty when metrics are enabled")
		}
	}
	return nil
}
