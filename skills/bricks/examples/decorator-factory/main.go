package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"go.uber.org/fx"
)

type QuoteInput struct {
	Quantity int
}

type QuoteUseCase struct{}

func (u *QuoteUseCase) Execute(ctx context.Context, input QuoteInput) (int, error) {
	if input.Quantity < 1 {
		return 0, fmt.Errorf("quantity must be positive")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return input.Quantity * 125, nil
}

type quoteTranslator struct{}

func (quoteTranslator) TranslateError(err error) error {
	return fmt.Errorf("quote unavailable: %w", err)
}

func quantity() (int, error) {
	if len(os.Args) < 2 {
		return 2, nil
	}
	return strconv.Atoi(os.Args[1])
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func provideLogger(lifecycle fx.Lifecycle) (logger.Logger, error) {
	cfg := logger.DefaultConfig()
	cfg.Level = "debug"
	cfg.OutputPaths = []string{"decorator.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		return log.Sync()
	}})
	return log, nil
}

func provideQuote(factory *ucdecorator.Factory) ucdecorator.UseCase[QuoteInput, int] {
	return ucdecorator.Wrap[QuoteInput, int](factory, &QuoteUseCase{})
}

func run() error {
	count, err := quantity()
	if err != nil {
		return err
	}
	var quoted ucdecorator.UseCase[QuoteInput, int]
	app := fx.New(
		ucdecorator.Module,
		metrics.Module,
		fx.Provide(
			provideLogger,
			func() ucdecorator.ErrorTranslator { return quoteTranslator{} },
			provideQuote,
		),
		fx.Populate(&quoted),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err == nil {
		var total int
		total, err = quoted.Execute(context.Background(), QuoteInput{Quantity: count})
		if err == nil {
			fmt.Printf("total_cents=%d\n", total)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}
