package validator

import "go.uber.org/fx"

// Module provides a pre-configured Validator instance with English translations.
// Ready to use with Uber FX dependency injection.
//
// Example:
//
//	fx.New(
//	    validator.Module,
//	    fx.Invoke(func(v validator.Validator) {
//	        // Use validator here
//	    }),
//	)
var Module = fx.Module(
	"validator",
	fx.Provide(New),
)
