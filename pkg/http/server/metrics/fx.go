package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

// LifecycleParams contains the dependencies for the metrics listener.
type LifecycleParams struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
	Config     config.Config[Config]
	Gatherer   prometheus.Gatherer
	Logger     *slog.Logger `optional:"true"`
}

// NewWithLifecycle creates a metrics listener managed by Fx.
func NewWithLifecycle(params LifecycleParams) (*Server, error) {
	cfg := params.Config.Get()
	if cfg.ReadHeaderTimeout == 0 {
		cfg.ReadHeaderTimeout = 10 * time.Second
	}

	server, err := New(cfg, params.Gatherer)
	if err != nil {
		return nil, err
	}

	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if startErr := server.StartContext(ctx); startErr != nil {
				return startErr
			}

			if params.Config.Get().Enabled {
				go func() {
					if serveErr, ok := <-server.Errors(); ok {
						logger.ErrorContext(ctx, "metrics server failed", "error", serveErr)

						if shutdownErr := params.Shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
							logger.ErrorContext(ctx, "request shutdown", "error", shutdownErr)
						}
					}
				}()
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	return server, nil
}

// Module starts an independent metrics listener configured under app.metrics.
// Applications supply a prometheus.Gatherer, typically their own Registry.
var Module = fx.Module("metrics-listener",
	config.Provide[Config]("app.metrics"),
	fx.Provide(NewWithLifecycle),
	fx.Invoke(func(*Server) {}),
)
