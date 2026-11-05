# Validation

Use `github.com/cristiano-pacheco/bricks/pkg/validator` to validate decoded application input. JSON decoding checks representation, not application rules. Call validation explicitly after decoding and before executing the operation. This package does not parse requests or select HTTP status codes.

## Construction and operations

`New() (Validator, error)` builds a go-playground engine with required-struct validation enabled and English translations registered. Handle construction errors. A missing English translator returns `translator not found`; translation-registration errors propagate. There is no configuration file, lifecycle hook, or shutdown operation.

Prefer `validator.Module` in an Fx application. It provides `validator.Validator`; consumers must request that interface. A wiring fragment follows, with `fx` imported from `go.uber.org/fx`:

```go
app := fx.New(
	validator.Module,
	fx.Invoke(func(inputValidator validator.Validator) error {
		return inputValidator.ValidateVar("person@example.com", "required,email")
	}),
)
```

Direct-construction fragment:

```go
inputValidator, err := validator.New()
if err != nil {
	return err
}
type Input struct {
	Name  string `validate:"required,min=3"`
	Email string `validate:"required,email"`
}
err = inputValidator.Validate(Input{Name: "Ada", Email: "ada@example.com"})
```

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

For a complete Fx program with a configured provider, alias, custom rule, custom English translation, scalar and struct operations, and owned start/stop, read [the validation example](../../examples/validation/README.md). Load it only when implementing that complete setup. Short snippets here are fragments, not standalone programs. The complete example is unexecuted; compilation and runtime checks were waived for this documentation ticket.
