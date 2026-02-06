package logger

import (
	"context"

	"go.uber.org/fx"
)

// Module provides logger dependencies for Uber FX
var Module = fx.Module("logger",
	fx.Provide(
		fx.Annotate(
			ProvideLogger,
			fx.As(new(Logger)),
		),
	),
)

// ProvideLogger creates a logger instance for FX dependency injection
func ProvideLogger(config Config) (*ZapLogger, error) {
	return New(config)
}

// Params defines the parameters for logger creation in FX
type Params struct {
	fx.In

	Config Config
}

// Result defines what the logger module provides to FX
type Result struct {
	fx.Out

	Logger Logger
}

// ProvideWithParams creates a logger with FX parameter injection
func ProvideWithParams(p Params) (Result, error) {
	logger, err := New(p.Config)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Logger: logger,
	}, nil
}

// ProvideWithLifecycle creates a logger and registers shutdown hook
func ProvideWithLifecycle(lc fx.Lifecycle, config Config) (*ZapLogger, error) {
	logger, err := New(config)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return logger.Sync()
		},
	})

	return logger, nil
}

// ModuleWithLifecycle provides logger with lifecycle management
var ModuleWithLifecycle = fx.Module("logger",
	fx.Provide(
		fx.Annotate(
			ProvideWithLifecycle,
			fx.As(new(Logger)),
		),
	),
)
