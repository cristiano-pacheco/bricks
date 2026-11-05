# Run the validation example

This application calculates a product quote. Fx injects the Bricks validator into a use case, and the use case rejects invalid input before calculating the total. No YAML or external service is required.

## 1. Copy both files

In a consumer module that already depends on Bricks and Fx, create an empty executable directory and copy these files into it:

- [product_quote_usecase.go](product_quote_usecase.go) defines the input tags, constructor, and `Execute` method.
- [main.go](main.go) provides the validator, constructs the use case, calls it, and stops the application.

Both files use `package main`, so you do not need to change local imports. For an inline walkthrough of both files, read the [validation guide](../../references/validator/usage.md).

## 2. Run the application

From the directory containing both copied files, run:

```sh
go run .
```

From a Bricks checkout, run this instead:

```sh
go run ./skills/bricks/examples/validation
```

The expected application output is:

```text
5000
```

Fx may also log lifecycle messages. The input is a unit price of `2500` cents and a quantity of `2`.

## 3. Try an invalid quantity

In `main.go`, change `Quantity: 2` to `Quantity: 0` and run the command again.

The first action in `Execute` is `uc.validator.Validate(input)`. The quantity fails its validation tags, so the method returns an empty output and an upstream validation error. `run` returns the error after deferred shutdown. `main` prints it to standard error and exits with status `1`. An application should handle validation errors according to its transport contract.

## 4. Adapt the pattern

Keep `validator.Module` once in your Fx composition. Inject `validator.Validator` into the use-case constructor and validate at the start of `Execute`. The transport should decode the request and call the use case. Application validation does not belong in lifecycle hooks.

In an application, move the use case into its module's `usecase` package and register its constructor in that module's Fx composition. The input tags set positive upper bounds for price and quantity; those bounds also keep multiplication within `int64`.

For scalar validation, translated messages, aliases, or custom rules, read the corresponding sections in the [validation guide](../../references/validator/usage.md). They are optional extensions, not setup requirements for this example.
