package main

import (
	"context"

	"github.com/cristiano-pacheco/bricks/pkg/validator"
)

type ProductQuoteInput struct {
	UnitPriceInCents int64 `validate:"required,gt=0,lte=1000000000"`
	Quantity         int64 `validate:"required,gt=0,lte=1000"`
}

type ProductQuoteOutput struct {
	TotalInCents int64
}

type ProductQuoteUseCase struct {
	validator validator.Validator
}

func NewProductQuoteUseCase(validator validator.Validator) *ProductQuoteUseCase {
	return &ProductQuoteUseCase{validator: validator}
}

func (uc *ProductQuoteUseCase) Execute(_ context.Context, input ProductQuoteInput) (ProductQuoteOutput, error) {
	if err := uc.validator.Validate(input); err != nil {
		return ProductQuoteOutput{}, err
	}

	return ProductQuoteOutput{
		TotalInCents: input.UnitPriceInCents * input.Quantity,
	}, nil
}
