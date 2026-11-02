package metrics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config controls the separate administrative metrics listener.
type Config struct {
	Enabled           bool          `config:"enabled"`
	Address           string        `config:"address"`
	Port              uint          `config:"port"`
	ReadTimeout       time.Duration `config:"readtimeout"`
	ReadHeaderTimeout time.Duration `config:"readheadertimeout"`
	WriteTimeout      time.Duration `config:"writetimeout"`
	IdleTimeout       time.Duration `config:"idletimeout"`
}

// Server owns the metrics listener and serves the supplied gatherer.
type Server struct {
	config   Config
	server   *http.Server
	mu       sync.RWMutex
	address  string
	listener net.Listener
	stopping atomic.Bool
	errors   chan error
}

// New creates a metrics server. Collectors can be registered with the registerer
// associated with gatherer before or after construction.
func New(config Config, gatherer prometheus.Gatherer) (*Server, error) {
	if config.Port > 65535 {
		return nil, fmt.Errorf("invalid metrics port: %d", config.Port)
	}
	if gatherer == nil {
		return nil, errors.New("metrics gatherer is required")
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	return &Server{
		config: config,
		server: &http.Server{
			Handler:           mux,
			ReadTimeout:       config.ReadTimeout,
			ReadHeaderTimeout: config.ReadHeaderTimeout,
			WriteTimeout:      config.WriteTimeout,
			IdleTimeout:       config.IdleTimeout,
		},
		errors: make(chan error, 1),
	}, nil
}

// Start binds before returning so startup failures are visible to the caller.
func (s *Server) Start() error {
	if !s.config.Enabled {
		return nil
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(s.config.Address, strconv.Itoa(int(s.config.Port))))
	if err != nil {
		return fmt.Errorf("bind metrics listener: %w", err)
	}
	s.mu.Lock()
	s.address = listener.Addr().String()
	s.listener = listener
	s.mu.Unlock()
	go func() {
		defer close(s.errors)
		if serveErr := s.server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && !s.stopping.Load() {
			s.errors <- fmt.Errorf("serve metrics: %w", serveErr)
		}
	}()
	return nil
}

// Shutdown stops the listener within the supplied deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	s.stopping.Store(true)
	err := s.server.Shutdown(ctx)
	if err != nil {
		err = errors.Join(err, s.server.Close())
	}
	s.mu.RLock()
	listener := s.listener
	s.mu.RUnlock()
	if listener != nil {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}
	return err
}

// Addr returns the effective listening address, or an empty string before start.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.address
}

// Errors reports unexpected serving failures.
func (s *Server) Errors() <-chan error {
	return s.errors
}
