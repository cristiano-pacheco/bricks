package chi

import (
	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
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
//	fx.New(chi.Module).Run()
//
// The module loads app.http from the application's configuration directory.
// See config/config.yaml for a configuration skeleton.
var Module = fx.Module(
	"httpserver-chi",
	config.Provide[Config]("app.http"),
	fx.Provide(NewWithLifecycle),
	fx.Invoke(func(*Server) {}), // Force server construction
)
