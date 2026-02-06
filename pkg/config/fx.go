package config

import (
	"go.uber.org/fx"
)

// Module provides config with fx lifecycle
var Module = fx.Module("config",
	fx.Provide(NewWithFx),
)

// ProvideConfig provides a typed config using generics for FX.
// Use this to inject your config struct directly into FX.
//
// Example:
//
//	fx.New(
//	    config.ProvideConfigDir("./config"),
//	    config.ProvideEnvironment("local"),
//	    config.ProvideConfig[AppConfig](),
//	    fx.Invoke(func(cfg AppConfig) {
//	        // use cfg
//	    }),
//	)
func ProvideConfig[T any]() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(cfg *Config) (T, error) {
				var result T
				if err := cfg.Unmarshal(&result); err != nil {
					return result, err
				}
				return result, nil
			},
		),
	)
}

// Params for dependency injection
type Params struct {
	fx.In
	ConfigDir   string `name:"config_dir"  optional:"true"`
	Environment string `name:"environment" optional:"true"`
}

// Result for dependency injection
type Result struct {
	fx.Out
	Config *Config
}

// NewWithFx creates a new config with fx lifecycle management
func NewWithFx(params Params) (Result, error) {
	configDir := params.ConfigDir
	if configDir == "" {
		configDir = "./config"
	}

	environment := params.Environment
	if environment == "" {
		environment = getEnvironment()
	}

	cfg, err := New(configDir, environment)
	if err != nil {
		return Result{}, err
	}

	return Result{Config: cfg}, nil
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

// ProvideEnvironment provides the environment for FX
func ProvideEnvironment(env string) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() string { return env },
			fx.ResultTags(`name:"environment"`),
		),
	)
}
