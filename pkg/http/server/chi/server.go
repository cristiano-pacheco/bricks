package chi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
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

// New creates a new HTTP server with Chi router.
func New(cfg Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
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

	// Add health check endpoint
	router.Get(healthCheckPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Create metrics server
	metricsRouter := chi.NewRouter()
	metricsRouter.Handle(metricsPath, promhttp.Handler())

	metricsServer := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", cfg.MetricsPort),
		Handler:      metricsRouter,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		server:        srv,
		router:        router,
		metricsServer: metricsServer,
		config:        cfg,
	}, nil
}

// NewWithLifecycle creates a new HTTP server with fx.Lifecycle management.
// The server is automatically started on application start and gracefully shut down on stop.
func NewWithLifecycle(cfg Config, lc fx.Lifecycle) (*Server, error) {
	server, err := New(cfg)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					// Log error but don't crash - fx will handle this
					_ = err
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx := ctx
			if server.config.ShutdownTimeout > 0 {
				var cancel context.CancelFunc
				shutdownCtx, cancel = context.WithTimeout(ctx, server.config.ShutdownTimeout)
				defer cancel()
			}
			return server.Shutdown(shutdownCtx)
		},
	})

	return server, nil
}

// Router returns the Chi router for registering routes.
func (s *Server) Router() *chi.Mux {
	return s.router
}

// Start begins listening and serving HTTP requests.
// Starts the metrics server on a separate port.
func (s *Server) Start() error {
	// Start metrics server
	go func() {
		if err := s.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = err
		}
	}()

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
// Also shuts down the metrics server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Shutdown metrics server
	if err := s.metricsServer.Shutdown(ctx); err != nil {
		_ = err
	}

	return s.server.Shutdown(ctx)
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.server.Addr
}

// MetricsAddr returns the metrics server address.
func (s *Server) MetricsAddr() string {
	return s.metricsServer.Addr
}
