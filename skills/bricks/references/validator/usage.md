# Validate use-case input

This guide builds a product quote use case that rejects invalid prices and quantities before calculating a total. The transport decodes input. The use case checks application rules at the start of `Execute`.

The validator needs no YAML, external service, or shutdown hook. Its rules live in Go struct tags.

## 1. Create the use case

Create `product_quote_usecase.go` in an executable directory of a module that already depends on Bricks and Fx:

```go
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
```

Both fields must be positive and within their tagged upper bounds. These bounds also keep the multiplication within `int64`. Invalid input returns an empty output and an error before the calculation.

## 2. Provide the validator and call the use case

Create `main.go` beside the first file:

```go
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	var quote *ProductQuoteUseCase
	app := fx.New(
		validator.Module,
		fx.Provide(NewProductQuoteUseCase),
		fx.Populate(&quote),
	)

	if err := app.Err(); err != nil {
		return fmt.Errorf("construct validation application: %w", err)
	}
	defer func() {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelStop()
		if err := app.Stop(stopCtx); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("stop validation application: %w", err))
		}
	}()

	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()
	if startErr != nil {
		return fmt.Errorf("start validation application: %w", startErr)
	}

	output, err := quote.Execute(context.Background(), ProductQuoteInput{
		UnitPriceInCents: 2500,
		Quantity:         2,
	})
	if err != nil {
		return err
	}
	fmt.Println(output.TotalInCents)
	return nil
}
```

Install `validator.Module` once. It provides `validator.Validator`; the use-case constructor receives that interface. `fx.Populate` requests the use case so Fx constructs it and its dependencies. Keep application validation in `Execute`, not in Fx lifecycle hooks.

## 3. Run the program and try invalid input

Run `go run .` from the directory containing both files. The expected application output is `5000`. Fx may also log lifecycle messages.

Change `Quantity` to `0` and run it again. `Execute` should return a validation error instead of a total. The deferred shutdown runs before `main` prints the error to standard error and exits with status `1`. An application caller should handle validation errors according to its transport contract.

The [distributed validation example](../../examples/validation/README.md) contains the same two-file application pattern. The supplied example also exercises Fx construction, startup, and shutdown. Custom rules and translation need their own checks; this happy-path program does not exercise them.

## Without Fx

Call `validator.New()`, check its error, then pass the result to `NewProductQuoteUseCase`. Use the same `Execute` method. This fragment replaces construction through Fx:

```go
inputValidator, err := validator.New()
if err != nil {
	return err
}
quote := NewProductQuoteUseCase(inputValidator)
```

The fragment belongs inside a function returning `error`. `New()` returns `(Validator, error)`. It creates a go-playground engine with required-struct validation enabled and registers English translations. A missing English translator returns `translator not found`; translation-registration errors propagate.

## Choose struct or scalar validation

| Operation | Use it for |
| --- | --- |
| `Validate(input)` or `Struct(input)` | A struct with validation tags. These methods are identical. |
| `ValidateVar(value, tags)` or `Var(value, tags)` | A scalar, such as `inputValidator.Var(17, "gte=18")`. These methods are identical. |
| `Engine()` | The shared `*validator/v10.Validate` engine for extensions. |
| `Translator()` | The configured English translator for rendering errors. |

Validation returns nil on success. It does not trim or normalize values.

The engine checks exported fields and recursively validates nested structs. Use `required` on a nested value struct when its zero value must fail. Required-struct mode does not make every untagged field mandatory. Use `dive` for slice or map elements, not ordinary nested structs. For example:

```go
type ProductInput struct {
	Tags []string `validate:"required,min=1,dive,required"`
}
```

This requires a nonempty slice whose elements are nonempty strings.

## Handle and translate errors

The methods return upstream errors unchanged. They do not return translated strings. Invalid values produce `validator/v10.ValidationErrors`. Invalid struct arguments, such as nil or a scalar passed to `Struct`, produce `*validator/v10.InvalidValidationError`. Unknown or malformed tags can panic; treat them as programming errors.

Use this fragment after `err := inputValidator.Validate(input)`. It requires standard-library `errors` and `fmt`, plus `playground "github.com/go-playground/validator/v10"`. It belongs inside a function returning `error`:

```go
var failures playground.ValidationErrors
var invalid *playground.InvalidValidationError
switch {
case err == nil:
	// Continue with valid input.
case errors.As(err, &failures):
	fmt.Println(failures.Translate(inputValidator.Translator()))
	return err
case errors.As(err, &invalid):
	return fmt.Errorf("invalid validation argument: %w", err)
default:
	return err
}
```

Each failure exposes `Field`, `Namespace`, `Tag`, `ActualTag`, and `Param`. Call `failure.Translate(inputValidator.Translator())` for one message. Names are Go field names unless you register a tag-name function.

The English translator does not select a language from HTTP headers. It is independent of Bricks i18n message and structured-error translation.

## Add aliases and custom rules

Register extensions once during construction, before validation or concurrent use. Registration mutates the shared engine.

This fragment requires `strings` and `playground "github.com/go-playground/validator/v10"`. It assumes `inputValidator` already exists and belongs inside a function returning `error`:

```go
engine := inputValidator.Engine()
engine.RegisterAlias("account_email", "required,email")
if err := engine.RegisterValidation("sku", func(field playground.FieldLevel) bool {
	return strings.HasPrefix(field.Field().String(), "SKU-")
}); err != nil {
	return err
}
```

Use `validate:"account_email"` for the alias or `validate:"sku"` on a string field. The `sku` rule is string-only. After registration, `inputValidator.Var("bad", "sku")` returns a validation error.

To translate a custom failure, also call the engine's `RegisterTranslation`. Supply the tag, translator, a registration callback using `translator.Add`, and a rendering callback using `translator.T`. Handle registration errors. If rendering fails, return the original field error text.

Other supported engine extension points are `RegisterStructValidation` for cross-field rules, `RegisterCustomTypeFunc` for extracting custom-type values, and `RegisterTagNameFunc` for client-facing field names. These are upstream engine APIs, not additional Bricks wrappers. Keep rules free of I/O. The Bricks interface has no context-taking validation methods; use `Engine()` explicitly when you need the engine's context-aware operations.
