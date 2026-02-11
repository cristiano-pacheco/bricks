package response

import "go.uber.org/fx"

var Module = fx.Module(
	"response",
	fx.Provide(
		fx.Annotate(
			NewErrorHandler,
			fx.As(new(ErrorHandler)),
		),
	),
)
