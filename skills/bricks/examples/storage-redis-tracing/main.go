package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	"go.uber.org/fx"
)

func underlying(client *bricksredis.Client) bricksredis.UniversalClient {
	return client.UniversalClient()
}

func registerWork(lc fx.Lifecycle, client bricksredis.UniversalClient) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		ctx, span := brickstrace.Span(ctx, "check-redis")
		defer span.End()
		value, err := client.Ping(ctx).Result()
		if err != nil {
			return fmt.Errorf("ping Redis: %w", err)
		}
		fmt.Println("redis=" + value)
		return nil
	}})
}

func run() error {
	app := fx.New(
		brickstrace.Module,
		brickstrace.RedisModule(),
		bricksredis.ClientModule,
		fx.Provide(underlying),
		fx.Invoke(registerWork),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
