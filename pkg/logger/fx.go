package logger

import (
	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"logger",
	config.Provide[Config]("app.logger"),
	fx.Provide(
		NewWithLifecycle,
		fx.Annotate(New, fx.As(new(Logger))),
	),
)
