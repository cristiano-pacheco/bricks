package database

import (
	"context"
	"time"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides database connection with fx lifecycle
var Module = fx.Module("database",
	fx.Provide(NewWithFx),
)

// Params for dependency injection
type Params struct {
	fx.In
	Config  Config
	Options []Option `optional:"true"`
}

// NewWithFx creates a new database client with fx lifecycle management
func NewWithFx(lc fx.Lifecycle, params Params) (*gorm.DB, error) {
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

	return client.DB(), nil
}

// ClientModule provides the full Client instance instead of just *gorm.DB
var ClientModule = fx.Module("database-client",
	fx.Provide(NewClientWithFx),
)

// NewClientWithFx creates a new database client with fx lifecycle management
func NewClientWithFx(lc fx.Lifecycle, params Params) (*Client, error) {
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
