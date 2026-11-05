package ucdecorator

import (
	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

var Module = fx.Module(
	"ucdecorator",
	config.Provide[Config]("app.ucdecorator"),
	fx.Provide(
		fx.Annotate(
			NewFactory,
			fx.ParamTags(``, ``, ``, ``),
		),
	),
)
