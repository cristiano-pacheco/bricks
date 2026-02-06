package redis

import (
	"context"
	"time"

	"go.uber.org/fx"
)

// Module provides Redis client with fx lifecycle
var Module = fx.Module("redis",
	fx.Provide(NewWithFx),
)

// Params for dependency injection
type Params struct {
	fx.In
	Config  Config
	Options []Option `optional:"true"`
}

// NewWithFx creates a new Redis client with fx lifecycle management
func NewWithFx(lc fx.Lifecycle, params Params) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewClient(ctx, params.Config, params.Options...)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return client.Ping(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})

	return client, nil
}

// UniversalClientModule provides the UniversalClient interface instead of *Client
var UniversalClientModule = fx.Module("redis-universal",
	fx.Provide(NewUniversalClientWithFx),
)

// NewUniversalClientWithFx creates a new Redis universal client with fx lifecycle management
func NewUniversalClientWithFx(lc fx.Lifecycle, params Params) (UniversalClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewClient(ctx, params.Config, params.Options...)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return client.Ping(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})

	return client.UniversalClient(), nil
}
