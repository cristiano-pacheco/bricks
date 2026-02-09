package validator

import "go.uber.org/fx"

// Module provides a pre-configured Validator instance with English translations.
// Ready to use with Uber FX dependency injection.
var Module = fx.Module(
	"validator",
	fx.Provide(
		fx.Annotate(
			New,
			fx.As(new(Validator)),
		),
	),
)
