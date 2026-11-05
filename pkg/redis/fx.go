package redis

import (
	"context"
	"errors"
	"time"

	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

const defaultFxTimeout = 30 * time.Second

// Module provides the Redis UniversalClient with automatic config loading and fx lifecycle.
// Config is loaded automatically from "app.redis" in your config files.
// The client automatically:
// - Loads config from "app.redis"
// - Connects and pings Redis on application start
// - Closes the connection on application stop
//
// Usage in your application:
//
//	fx.New(
//	    redis.Module,
//	    fx.Invoke(func(redis.UniversalClient) {}),
//	)
var Module = fx.Module("redis",
	config.Provide[Config]("app.redis"),
	fx.Provide(configValue, NewUniversalClientWithFx),
)

// ClientModule provides the Redis *Client with automatic config loading and fx lifecycle.
// Use this instead of Module when you need access to bricks-specific helpers such as
// namespacing, metrics, or pool stats.
// The client automatically:
// - Loads config from "app.redis"
// - Connects and pings Redis on application start
// - Closes the connection on application stop
//
// Usage in your application:
//
//	fx.New(
//	    redis.ClientModule,
//	    fx.Invoke(func(*redis.Client) {}),
//	)
var ClientModule = fx.Module("redis-client",
	config.Provide[Config]("app.redis"),
	fx.Provide(configValue, NewWithFx),
)

func configValue(value config.Config[Config]) Config {
	return value.Get()
}

// TracingEnabled is supplied by the tracing module to validate framework-owned clients.
type TracingEnabled struct{}

// Params for dependency injection.
type Params struct {
	fx.In

	Config  Config
	Options []Option        `optional:"true"`
	Tracing *TracingEnabled `optional:"true"`
}

// NewWithFx creates a new Redis *Client with fx lifecycle management.
func NewWithFx(lc fx.Lifecycle, params Params) (*Client, error) {
	if params.Tracing != nil && params.Config.Type == ClientTypeCluster {
		return nil, errors.New("redis tracing: cluster clients are not supported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultFxTimeout)
	defer cancel()

	client, err := NewClient(ctx, params.Config, params.Options...)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return client.Ping(ctx)
		},
		OnStop: func(_ context.Context) error {
			return client.Close()
		},
	})

	return client, nil
}

// NewUniversalClientWithFx creates a new Redis UniversalClient with fx lifecycle management.
func NewUniversalClientWithFx(lc fx.Lifecycle, params Params) (UniversalClient, error) {
	if params.Tracing != nil && params.Config.Type == ClientTypeCluster {
		return nil, errors.New("redis tracing: cluster clients are not supported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultFxTimeout)
	defer cancel()

	client, err := NewClient(ctx, params.Config, params.Options...)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return client.Ping(ctx)
		},
		OnStop: func(_ context.Context) error {
			return client.Close()
		},
	})

	return client.UniversalClient(), nil
}
