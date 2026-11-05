package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"go.uber.org/fx"
)

func registerOrder(lifecycle fx.Lifecycle) {
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return createOrder(ctx)
	}})
}

func createOrder(ctx context.Context) error {
	ctx, span := brickstrace.Span(ctx, "create-order")
	defer span.End()
	return ctx.Err()
}

func run() error {
	app := fx.New(brickstrace.Module, fx.Invoke(registerOrder))
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	err := app.Start(startCtx)
	cancelStart()
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
