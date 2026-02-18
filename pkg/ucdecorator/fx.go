package ucdecorator

import (
	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"ucdecorator",
	config.Provide[Config]("app.ucdecorator"),
	fx.Provide(
		fx.Annotate(
			NewFactory,
			fx.ParamTags(``, ``, ``, `optional:"true"`),
		),
	),
)
