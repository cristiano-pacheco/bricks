package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/validator"
	"go.uber.org/fx"
)

func run() (err error) {
	var productQuoteUseCase *ProductQuoteUseCase
	app := fx.New(
		validator.Module,
		fx.Provide(NewProductQuoteUseCase),
		fx.Populate(&productQuoteUseCase),
	)

	if err := app.Err(); err != nil {
		return fmt.Errorf("construct validation application: %w", err)
	}
	defer func() {
		stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelStop()
		if stopErr := app.Stop(stopContext); stopErr != nil {
			err = errors.Join(err, fmt.Errorf("stop validation application: %w", stopErr))
		}
	}()

	startContext, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	startErr := app.Start(startContext)
	cancelStart()
	if startErr != nil {
		return fmt.Errorf("start validation application: %w", startErr)
	}

	output, err := productQuoteUseCase.Execute(context.Background(), ProductQuoteInput{
		UnitPriceInCents: 2500,
		Quantity:         2,
	})
	if err != nil {
		return fmt.Errorf("execute product quote: %w", err)
	}
	fmt.Println(output.TotalInCents)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
