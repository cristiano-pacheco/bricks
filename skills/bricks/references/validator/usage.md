# Validation

Use `github.com/cristiano-pacheco/bricks/pkg/validator` to validate application input structs at the start of a use case's `Execute` method. JSON decoding checks representation, not application rules. The transport decodes the request and calls the use case. The use case validates its input before performing business work.

## Construction and operations

`New() (Validator, error)` builds a go-playground engine with required-struct validation enabled and English translations registered. Handle construction errors. A missing English translator returns `translator not found`; translation-registration errors propagate. There is no configuration file, lifecycle hook, or shutdown operation.

Use `validator.Module` once in the Fx process composition. Inject `validator.Validator` into the use case through its constructor. Do not replace the module with a custom validator provider or run application validation in lifecycle hooks.

Composition fragment, with `fx` imported from `go.uber.org/fx`:

```go
app := fx.New(
	validator.Module,
	fx.Provide(NewProductQuoteUseCase),
)
```

Use-case fragment, with the standard-library `context` import:

```go
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
	return ProductQuoteOutput{TotalInCents: input.UnitPriceInCents * input.Quantity}, nil
}
```

For a caller without Fx, the supported direct alternative is `inputValidator, err := validator.New()`. Handle the error, then pass the instance to `NewProductQuoteUseCase`. This does not change where the use case validates its input.

`Validate` and `Struct` are identical struct-validation operations. `ValidateVar` and `Var` are identical scalar-validation operations, for example `inputValidator.Var(17, "gte=18")`. Validation returns nil on success and does not normalize or trim values.

The engine checks exported fields and recursively validates nested structs. Apply `required` to a nested value struct when its zero value must fail. Use `dive` for slice or map elements, not for ordinary nested structs. For example, `Tags []string` with `validate:"required,min=1,dive,required"` requires a nonempty slice and nonempty elements. Required-struct mode changes `required` on non-pointer structs; it does not make every untagged field mandatory.

## Errors and translations

Methods return the upstream error unchanged. They do not return translated strings despite some source comments saying otherwise. Invalid values produce `validator/v10.ValidationErrors`. Invalid struct arguments, such as nil or a scalar passed to `Struct`, produce `*validator/v10.InvalidValidationError`. Malformed or unknown validation tags can panic and are programming errors, not rejected user values.

Error-handling fragment, with `errors` from the standard library and `playground` imported from `github.com/go-playground/validator/v10`:

```go
var failures playground.ValidationErrors
var invalid *playground.InvalidValidationError
switch {
case err == nil:
	// Continue with valid input.
case errors.As(err, &failures):
	messages := failures.Translate(inputValidator.Translator())
	fmt.Println(messages)
case errors.As(err, &invalid):
	return fmt.Errorf("invalid validation argument: %w", err)
default:
	return err
}
```

Import `fmt` for the fragment. Individual failures expose `Field`, `Namespace`, `Tag`, `ActualTag`, and `Param`; `failure.Translate(inputValidator.Translator())` returns one message. Names are Go field names unless you register a tag-name function. English registration does not automatically select a language from HTTP headers. This translator is independent of Bricks i18n message and structured-error translation.

## Engine extension

`Engine()` returns the shared `*playground.Validate`; `Translator()` returns its English `ut.Translator`. Register extensions once during construction, before validation or concurrent use. Registration mutates the engine; it is not a per-request operation.

Alias and custom-rule fragment:

```go
engine := inputValidator.Engine()
engine.RegisterAlias("account_email", "required,email")
err = engine.RegisterValidation("sku", func(field playground.FieldLevel) bool {
	return strings.HasPrefix(field.Field().String(), "SKU-")
})
if err != nil {
	return err
}
err = inputValidator.Var("bad", "sku")
```

Import `strings`. The `sku` rule is string-only; apply it only to string fields. Register its message separately if callers need a translated custom failure. `RegisterTranslation` takes the tag, translator, a registration callback using `translator.Add`, and a rendering callback using `translator.T`. Handle registration errors and return the original field error text if rendering fails.

Other supported upstream extension points are `RegisterStructValidation` for cross-field acceptance, `RegisterCustomTypeFunc` for extracting values from custom types, and `RegisterTagNameFunc` for client-facing field names. These are engine APIs, not additional Bricks wrappers. Keep acceptance rules in the validator rather than performing I/O inside a rule. The Bricks interface has no context-taking validation methods; context-aware engine operations require explicit use of `Engine()`.

For a complete program with `validator.Module`, constructor injection, and struct validation at the start of `Execute`, read [the validation example](../../examples/validation/README.md). Load it only when a complete example is requested. Short snippets here are fragments, not standalone programs. The complete example is unexecuted; compilation and runtime checks were waived for this documentation ticket.
