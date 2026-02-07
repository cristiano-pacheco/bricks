package config

import (
	"go.uber.org/fx"
)

// ProvideConfig provides a typed config using generics for FX.
// Use this to inject your config struct directly into FX.
//
// Example:
//
//	fx.New(
//	    config.ProvideConfigDir("./config"),
//	    config.ProvideConfig[AppConfig](),
//	    fx.Invoke(func(cfg AppConfig) {
//	        // use cfg
//	    }),
//	)
func ProvideConfig[T any]() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(params Params) (T, error) {
				configDir := params.ConfigDir
				if configDir == "" {
					configDir = "./config"
				}
				cfg, err := New[T](configDir)
				if err != nil {
					var zero T
					return zero, err
				}
				return cfg.Get(), nil
			},
		),
	)
}

// Params for dependency injection
type Params struct {
	fx.In
	ConfigDir string `name:"config_dir" optional:"true"`
}

// ProvideConfigDir provides the config directory for FX
func ProvideConfigDir(dir string) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() string { return dir },
			fx.ResultTags(`name:"config_dir"`),
		),
	)
}
