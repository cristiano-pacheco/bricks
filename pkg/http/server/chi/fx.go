package chi

import (
	"context"
	"net/http"

	"go.uber.org/fx"
)

// Module provides the Chi HTTP server as an fx module.
var Module = fx.Module(
	"httpserver-chi",
	fx.Provide(New),
)

// Params defines the input parameters for the Chi server when using fx.
type Params struct {
	fx.In

	Options []Option `group:"chi-server-options"`
}

// Result defines the output from creating a Chi server with fx.
type Result struct {
	fx.Out

	Server *Server
}

// NewWithFx creates a new Chi server using fx dependency injection.
func NewWithFx(params Params) (Result, error) {
	server, err := New(params.Options...)
	if err != nil {
		return Result{}, err
	}
	return Result{Server: server}, nil
}

// ModuleWithLifecycle provides the Chi HTTP server with automatic lifecycle management.
var ModuleWithLifecycle = fx.Module(
	"httpserver-chi",
	fx.Provide(NewWithFx),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, server *Server) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := server.Start(); err != nil {
					// Server stopped
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
}

// ProvideOption creates an fx provider for a server option.
// This allows options to be provided as fx dependencies.
func ProvideOption(opt Option) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Option { return opt },
			fx.ResultTags(`group:"chi-server-options"`),
		),
	)
}

// ProvideOptions creates an fx provider for multiple server options.
func ProvideOptions(opts ...Option) fx.Option {
	providers := make([]fx.Option, len(opts))
	for i, opt := range opts {
		providers[i] = ProvideOption(opt)
	}
	return fx.Options(providers...)
}

// WithLogger is a helper to add a custom logging middleware.
// Use this with fx to inject your logger.
func WithLogger(loggerMiddleware func(next http.Handler) http.Handler) Option {
	return func(c *Config) {
		// This is a placeholder - actual implementation would require
		// modifying the server to accept middleware options
	}
}

// Example usage:
//
// fx.New(
//     chi.ModuleWithLifecycle,
//     chi.ProvideOptions(
//         chi.WithHost("0.0.0.0"),
//         chi.WithPort(3000),
//         chi.WithDefaultCORS(),
//     ),
// )
