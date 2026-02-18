package ucdecorator

import (
	"go.uber.org/fx"
)

var Module = fx.Module(
	"ucdecorator",
	fx.Provide(NewFactory),
)
