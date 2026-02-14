package chi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/samber/lo"
	httpSwagger "github.com/swaggo/http-swagger/v2"
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
	registry      *RouteRegistry
	logger        *slog.Logger
}

// New creates a new HTTP server with Chi router.
func New(cfg Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	logger := slog.Default()

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
		registry:      NewRouteRegistry(),
		logger:        logger,
	}, nil
}

// NewWithLifecycleParams contains dependencies for creating a server with lifecycle.
type NewWithLifecycleParams struct {
	fx.In
	Config config.Config[Config]
	LC     fx.Lifecycle
	Routes []Route      `group:"routes"`
	Logger *slog.Logger `               optional:"true"`
}

// NewWithLifecycle creates a new HTTP server with fx.Lifecycle management.
// The server is automatically started on application start and gracefully shut down on stop.
// All routes from the "routes" group are automatically registered and configured.
func NewWithLifecycle(params NewWithLifecycleParams) (*Server, error) {
	server, err := New(params.Config.Get())
	if err != nil {
		return nil, err
	}

	// Use injected logger if available, otherwise use default
	if params.Logger != nil {
		server.logger = params.Logger
	}

	// Register all routes from FX group
	server.RegisterRoutes(params.Routes)

	params.LC.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// Setup routes before starting
			server.SetupRoutes()

			// Start server in background
			go func() {
				if startErr := server.Start(); startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
					// Log error but don't crash - fx will handle this
					_ = startErr
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

// RegisterRoute adds a route to the server's registry.
func (s *Server) RegisterRoute(route Route) {
	s.registry.Add(route)
}

// RegisterRoutes adds multiple routes to the server's registry.
func (s *Server) RegisterRoutes(routes []Route) {
	for _, route := range routes {
		s.registry.Add(route)
	}
}

// SetupRoutes configures all registered routes on the server.
// This should be called before Start().
func (s *Server) SetupRoutes() {
	s.registry.SetupAll(s)

	// Add swagger after module routes
	if s.config.Swagger != nil && s.config.Swagger.Enabled {
		path := defaultSwaggerPath
		if !lo.IsEmpty(s.config.Swagger.Path) {
			path = s.config.Swagger.Path
			if !strings.HasSuffix(path, "/*") {
				path = path + "/*"
			}
		}
		s.router.Get(path, httpSwagger.WrapHandler)
	}

	// Always log routes on startup
	s.logRoutes()
}

// logRoutes logs all registered routes to stdout.
func (s *Server) logRoutes() {
	s.logServerRoutes(s.router, "HTTP Server", s.server.Addr)
}

// logMetricsRoutes logs all registered metrics routes to stdout.
func (s *Server) logMetricsRoutes() {
	if metricsRouter, ok := s.metricsServer.Handler.(*chi.Mux); ok {
		s.logServerRoutes(metricsRouter, "Metrics Server", s.metricsServer.Addr)
	}
}

// logServerRoutes logs routes for a given router with a custom title and address.
func (s *Server) logServerRoutes(router *chi.Mux, serverName, addr string) {
	s.logger.Info(fmt.Sprintf("%s: http://%s", serverName, addr))
	s.logger.Info(fmt.Sprintf("%s routes:", serverName))
	s.logger.Info("==================")

	walkFunc := s.createRouteWalkFunc()
	if err := chi.Walk(router, walkFunc); err != nil {
		s.logger.Error(fmt.Sprintf("Error walking %s routes: %v", serverName, err))
	}

	s.logger.Info("==================")
}

// createRouteWalkFunc creates a walk function for logging routes.
func (s *Server) createRouteWalkFunc() func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
	return func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// Skip internal chi routes
		if strings.HasPrefix(route, "/*") {
			return nil
		}

		// Format the route for better readability
		route = strings.ReplaceAll(route, "/*", "")
		if route == "" {
			route = "/"
		}

		s.logger.Info(fmt.Sprintf("  %-7s %s", method, route))
		return nil
	}
}

// Start begins listening and serving HTTP requests.
// Starts the metrics server on a separate port.
func (s *Server) Start() error {
	// Log metrics server routes
	s.logMetricsRoutes()

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
