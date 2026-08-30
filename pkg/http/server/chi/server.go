package chi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

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
	server          *http.Server
	router          *chi.Mux
	metricsServer   *http.Server
	config          Config
	registry        *RouteRegistry
	logger          *slog.Logger
	mu              sync.RWMutex
	listener        net.Listener
	metricsListener net.Listener
	started         bool
}

// New creates a new HTTP server with Chi router.
func New(cfg Config) (*Server, error) {
	cfg.Address = strings.TrimSpace(cfg.Address)
	cfg.MetricsAddress = strings.TrimSpace(cfg.MetricsAddress)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	logger := slog.Default()
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	if cfg.EnableRequestLogging {
		router.Use(middleware.Logger)
	}
	router.Use(middleware.Recoverer)

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

	router.Get(healthCheckPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:         cfg.listenAddress(),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	var metricsServer *http.Server
	if cfg.EnableMetrics {
		metricsRouter := chi.NewRouter()
		metricsRouter.Handle(metricsPath, promhttp.Handler())
		metricsServer = &http.Server{
			Addr:         cfg.metricsListenAddress(),
			Handler:      metricsRouter,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		}
	}

	return &Server{
		server:        server,
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

	if params.Logger != nil {
		server.logger = params.Logger
	}

	server.RegisterRoutes(params.Routes)

	params.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			server.SetupRoutes()
			return server.startAsync(ctx)
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
// Call it before Start when using the server without Fx.
func (s *Server) SetupRoutes() {
	s.registry.SetupAll(s)

	if s.config.Swagger != nil && s.config.Swagger.Enabled {
		path := defaultSwaggerPath
		if !lo.IsEmpty(s.config.Swagger.Path) {
			path = s.config.Swagger.Path
			if !strings.HasSuffix(path, "/*") {
				path += "/*"
			}
		}
		s.router.Get(path, httpSwagger.WrapHandler)
	}
}

// Start binds the configured listeners and serves the main listener until it is shut down.
// Use Shutdown to stop the server.
func (s *Server) Start() error {
	listener, metricsListener, err := s.bind(context.Background())
	if err != nil {
		return err
	}

	if metricsListener != nil {
		go s.serve(s.metricsServer, metricsListener)
	}
	return s.server.Serve(listener)
}

// Shutdown gracefully shuts down every enabled listener.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil
	}
	servers := []*http.Server{s.server, s.metricsServer}
	listeners := []net.Listener{s.listener, s.metricsListener}
	s.mu.RUnlock()

	shutdownErrors := make(chan error, len(servers))
	var waitGroup sync.WaitGroup
	for _, server := range servers {
		if server == nil {
			continue
		}
		waitGroup.Add(1)
		go func(server *http.Server) {
			defer waitGroup.Done()
			shutdownErrors <- server.Shutdown(ctx)
		}(server)
	}
	waitGroup.Wait()
	close(shutdownErrors)

	for _, listener := range listeners {
		if listener != nil {
			_ = listener.Close()
		}
	}

	s.mu.Lock()
	s.listener = nil
	s.metricsListener = nil
	s.mu.Unlock()

	var errs []error
	for err := range shutdownErrors {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Addr returns the configured address before startup and the effective listener address after startup.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.server.Addr
}

// MetricsAddr returns the configured or effective metrics address.
// It returns an empty string when metrics are disabled.
func (s *Server) MetricsAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.metricsListener != nil {
		return s.metricsListener.Addr().String()
	}
	if s.metricsServer == nil {
		return ""
	}
	return s.metricsServer.Addr
}

func (s *Server) startAsync(ctx context.Context) error {
	listener, metricsListener, err := s.bind(ctx)
	if err != nil {
		return err
	}

	if metricsListener != nil {
		go s.serve(s.metricsServer, metricsListener)
	}
	go s.serve(s.server, listener)
	return nil
}

func (s *Server) bind(ctx context.Context) (net.Listener, net.Listener, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil, nil, ErrServerStarted
	}

	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", s.server.Addr)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s: %w", ErrListen, s.server.Addr, err)
	}

	var metricsListener net.Listener
	if s.metricsServer != nil {
		metricsListener, err = listenConfig.Listen(ctx, "tcp", s.metricsServer.Addr)
		if err != nil {
			_ = listener.Close()
			return nil, nil, fmt.Errorf("%w: %s: %w", ErrListen, s.metricsServer.Addr, err)
		}
	}

	s.listener = listener
	s.metricsListener = metricsListener
	s.started = true
	s.server.Addr = listener.Addr().String()
	if metricsListener != nil {
		s.metricsServer.Addr = metricsListener.Addr().String()
	}

	if s.config.EnableRouteLogging {
		s.logServerRoutes(s.router, "HTTP Server", s.server.Addr)
		if metricsListener != nil {
			if metricsRouter, ok := s.metricsServer.Handler.(*chi.Mux); ok {
				s.logServerRoutes(metricsRouter, "Metrics Server", s.metricsServer.Addr)
			}
		}
	}

	return listener, metricsListener, nil
}

func (s *Server) serve(server *http.Server, listener net.Listener) {
	_ = server.Serve(listener)
}

func (s *Server) logServerRoutes(router *chi.Mux, serverName, addr string) {
	s.logger.Info(fmt.Sprintf("%s: http://%s", serverName, addr))
	s.logger.Info(fmt.Sprintf("%s routes:", serverName))
	s.logger.Info("==================")

	if err := chi.Walk(router, s.createRouteWalkFunc()); err != nil {
		s.logger.Error(fmt.Sprintf("Error walking %s routes: %v", serverName, err))
	}

	s.logger.Info("==================")
}

func (s *Server) createRouteWalkFunc() func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
	return func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/*") {
			return nil
		}

		route = strings.ReplaceAll(route, "/*", "")
		if route == "" {
			route = "/"
		}

		s.logger.Info(fmt.Sprintf("  %-7s %s", method, route))
		return nil
	}
}
