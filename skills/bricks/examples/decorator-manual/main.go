package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
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

func run() (err error) {
	count, err := quantity()
	if err != nil {
		return err
	}
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"decorator.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, log.Sync()) }()

	measurements, err := metrics.NewPrometheusUseCaseMetrics()
	if err != nil {
		return err
	}
	quoted := ucdecorator.Chain[QuoteInput, int](
		&QuoteUseCase{}, log, measurements, quoteTranslator{},
		"quote", "QuoteUseCase.Execute",
	)
	total, err := quoted.Execute(context.Background(), QuoteInput{Quantity: count})
	if err != nil {
		return err
	}
	fmt.Printf("total_cents=%d\n", total)
	return nil
}
