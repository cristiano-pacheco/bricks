package chi

import (
	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.uber.org/fx"
)

// Module provides the Chi HTTP server with automatic lifecycle management.
// The server automatically:
// - Collects all routes from the "routes" FX group
// - Configures routes on startup
// - Starts the HTTP server
// - Gracefully shuts down on application stop
//
// Usage in your application:
//
//	fx.Module(
//	    "httpserver",
//	    config.Provide[chi.Config]("app.http"),
//	    chi.Module,
//	    fx.Invoke(func(*chi.Server) {}), // Force server construction
//	)
var Module = fx.Module(
	"httpserver-chi",
	config.Provide[Config]("app.http"),
	fx.Provide(NewWithLifecycle),
	fx.Invoke(func(*Server) {}), // Force server construction
)
