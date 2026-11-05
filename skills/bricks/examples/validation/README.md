# Validation example

Load this example when you need complete Fx wiring for a customized Bricks validator. For ordinary construction, struct and scalar rules, or error handling, read the [validation guide](../../references/validator/usage.md) instead.

[main.go](main.go) is a complete program with imports and explicit start and stop. The Fx provider calls `validator.New`, registers the `account_email` alias, registers the string-only `sku` rule, and adds its English translation before returning `validator.Validator`. The consumer demands that interface and exercises the operations in an OnStart hook. It accepts valid struct and scalar values, handles a custom-rule failure as upstream validation errors, translates it, and recognizes an invalid struct argument.

Use this configured provider instead of `validator.Module`, not alongside it. Both provide the same unnamed interface. There are no value groups, external services, configuration files, interface annotations, or additional providers to supply. Returning the interface directly makes `fx.As` unnecessary. Extensions finish before consumers can validate. The validator owns no shutdown resource; the program still stops the Fx application.

In a consumer module that already depends on Bricks and Fx, copy `main.go` into an otherwise empty executable directory and run `go run .`. Use the application's selected Bricks dependency. The example needs no framework checkout, external skills, or supporting repository files. From a Bricks checkout, the equivalent command is `go run ./skills/bricks/examples/validation`.

Expected application messages are `SKU must start with SKU-` and `validation example complete`. Fx may also log lifecycle messages. A construction, registration, validation, or startup failure stops the demonstration rather than being ignored.

This is complete runnable source, not an executed result. Compilation, Fx graph verification, runtime validation, translations, and lifecycle checks were waived by the user for ticket 04. The commands above are instructions for later verification, not evidence of a pass. No test suite was added or run.
