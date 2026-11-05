package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/validator"
	ut "github.com/go-playground/universal-translator"
	playground "github.com/go-playground/validator/v10"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(func() (validator.Validator, error) {
			inputValidator, err := validator.New()
			if err != nil {
				return nil, err
			}
			engine := inputValidator.Engine()
			engine.RegisterAlias("account_email", "required,email")
			if err := engine.RegisterValidation("sku", func(field playground.FieldLevel) bool {
				return strings.HasPrefix(field.Field().String(), "SKU-")
			}); err != nil {
				return nil, err
			}
			if err := engine.RegisterTranslation(
				"sku",
				inputValidator.Translator(),
				func(translator ut.Translator) error {
					return translator.Add("sku", "{0} must start with SKU-", true)
				},
				func(translator ut.Translator, field playground.FieldError) string {
					message, err := translator.T("sku", field.Field())
					if err != nil {
						return field.Error()
					}
					return message
				},
			); err != nil {
				return nil, err
			}
			return inputValidator, nil
		}),
		fx.Invoke(func(lifecycle fx.Lifecycle, inputValidator validator.Validator) {
			lifecycle.Append(fx.Hook{
				OnStart: func(context.Context) error {
					type Input struct {
						Email string `validate:"account_email"`
						SKU   string `validate:"required,sku"`
					}
					valid := Input{Email: "ada@example.com", SKU: "SKU-42"}
					if err := inputValidator.Validate(valid); err != nil {
						return err
					}
					if err := inputValidator.Struct(valid); err != nil {
						return err
					}
					if err := inputValidator.ValidateVar("ada@example.com", "account_email"); err != nil {
						return err
					}
					if err := inputValidator.Var("SKU-42", "sku"); err != nil {
						return err
					}
					var failures playground.ValidationErrors
					err := inputValidator.Validate(Input{Email: "ada@example.com", SKU: "bad"})
					if !errors.As(err, &failures) {
						return fmt.Errorf("expected custom-rule validation errors, got %v", err)
					}
					for _, failure := range failures {
						fmt.Println(failure.Translate(inputValidator.Translator()))
					}
					var invalid *playground.InvalidValidationError
					if err := inputValidator.Struct(nil); !errors.As(err, &invalid) {
						return fmt.Errorf("expected invalid validation argument, got %v", err)
					}
					fmt.Println("validation example complete")
					return nil
				},
			})
		}),
	)
	startContext, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStart()
	if err := app.Start(startContext); err != nil {
		panic(err)
	}
	stopContext, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	if err := app.Stop(stopContext); err != nil {
		panic(err)
	}
}
