# Validation example

Load this example when you need a complete Fx application that injects the Bricks validator into a use case. For the package contract and optional engine extensions, read the [validation guide](../../references/validator/usage.md).

[main.go](main.go) installs `validator.Module`, registers `NewProductQuoteUseCase`, and obtains the concrete use case through `fx.Populate`. It starts the application, calls `Execute` with a tagged input struct, prints the total, and stops the application. There is no custom validator provider or application validation in lifecycle hooks.

[product_quote_usecase.go](product_quote_usecase.go) declares the input, output, concrete use case, and constructor. The constructor receives `validator.Validator`. The first action in `Execute` is `uc.validator.Validate(input)`. Failure returns an empty output and the validation error before calculating the quote. The example's price and quantity bounds keep the multiplication within `int64`.

This follows the application pattern used by JBN and go-shortner. Validation belongs in the use case, not in the transport or Fx composition. The example keeps both files in `package main` so they can be copied without adapting local imports. In an application, put the use case in its module's `usecase` package and register its constructor in that module's Fx composition.

Copy both Go files into an otherwise empty executable directory in a consumer module that already depends on Bricks and Fx, then run `go run .`. No configuration files, external services, external skills, or framework checkout are needed. From a Bricks checkout, use `go run ./skills/bricks/examples/validation`.

The expected application output is `5000`. Fx may also log lifecycle messages. Inputs that violate the tags return upstream validation errors from `Execute`; the use case does not translate or replace them. Scalar validation and engine customization are separate supported APIs, not prerequisites for this pattern.

This is complete source, not an executed result. Compilation, Fx graph verification, runtime behavior, and lifecycle checks were waived by the user for ticket 04. The commands above are instructions for later verification, not evidence of a pass. No test suite was added or run.
