package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/redis"
)

func roundTrip(ctx context.Context, client *redis.Client) (err error) {
	raw := client.UniversalClient()

	key := client.WithNamespace("skill:example")
	if err := raw.Set(ctx, key, "hello", time.Minute).Err(); err != nil {
		return fmt.Errorf("set example: %w", err)
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = errors.Join(err, raw.Del(cleanupCtx, key).Err())
	}()

	value, err := raw.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("get example: %w", err)
	}

	if value != "hello" {
		return fmt.Errorf("unexpected value %q", value)
	}

	fmt.Printf("%s=%s\n", key, value)

	return nil
}

func registerOperation(lc fx.Lifecycle, client *redis.Client) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return roundTrip(ctx, client)
	}})
}

func run() error {
	app := fx.New(redis.ClientModule, fx.Invoke(registerOperation))
	if err := app.Err(); err != nil {
		return err
	}

	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)

	cancelStart()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStop()

	return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
