package logger

import (
	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

var Module = fx.Module(
	"logger",
	config.Provide[Config]("app.logger"),
	fx.Provide(
		fx.Annotate(NewWithLifecycle, fx.As(new(Logger))),
	),
)
