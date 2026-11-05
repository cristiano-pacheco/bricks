package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/validator"
	"go.uber.org/fx"
)

func main() {
	var productQuoteUseCase *ProductQuoteUseCase
	app := fx.New(
		validator.Module,
		fx.Provide(NewProductQuoteUseCase),
		fx.Populate(&productQuoteUseCase),
	)

	startContext, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStart()
	if err := app.Start(startContext); err != nil {
		panic(err)
	}
	defer func() {
		stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelStop()
		if err := app.Stop(stopContext); err != nil {
			panic(err)
		}
	}()

	output, err := productQuoteUseCase.Execute(context.Background(), ProductQuoteInput{
		UnitPriceInCents: 2500,
		Quantity:         2,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(output.TotalInCents)
}
