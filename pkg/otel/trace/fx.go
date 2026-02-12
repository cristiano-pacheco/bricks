package trace

import (
	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.uber.org/fx"
)

// Module is the fx module for OpenTelemetry tracing integration.
// Add this to your fx.App to enable distributed tracing.
var Module = fx.Module(
	"otel.trace",
	config.Provide[TracerConfig]("app.open-telemetry"),
	fx.Invoke(InitializeWithLifecycle),
)

// InitializeWithLifecycle configures and initializes OpenTelemetry tracing with fx.Lifecycle management.
// The tracer is automatically shut down when the application stops.
func InitializeWithLifecycle(lc fx.Lifecycle, cfg config.Config[TracerConfig]) error {
	err := Initialize(cfg.Get())
	if err != nil {
		return err
	}

	lc.Append(fx.Hook{
		OnStop: Shutdown,
	})

	return nil
}
