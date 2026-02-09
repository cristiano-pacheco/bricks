package chi

import "go.uber.org/fx"

// Module provides the Chi HTTP server as an fx module.
// Requires a Config to be provided in the fx app.
var Module = fx.Module(
	"httpserver-chi",
	fx.Provide(New),
)

// ModuleWithLifecycle provides the Chi HTTP server with automatic lifecycle management.
// Requires a Config to be provided in the fx app.
// The server is automatically started and gracefully shut down.
var ModuleWithLifecycle = fx.Module(
	"httpserver-chi",
	fx.Provide(NewWithLifecycle),
)

//     chi.ProvideOptions(
//         chi.WithHost("0.0.0.0"),
//         chi.WithPort(3000),
//         chi.WithDefaultCORS(),
//     ),
// )
