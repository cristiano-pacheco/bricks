package config

import (
	"go.uber.org/fx"
)

// Provide creates an fx.Provide option for loading config at the specified path.
// This helper reduces boilerplate when creating config providers.
//
// Example:
//
//	fx.Module("mymodule",
//	    config.Provide[MyConfig]("app.mymodule"),
//	)
func Provide[T any](path string) fx.Option {
	return fx.Provide(func() (Config[T], error) {
		return New[T](WithPath(path))
	})
}
