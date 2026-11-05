# Return a structured application error

Use `github.com/cristiano-pacheco/bricks/pkg/errs` when an application error needs a public code, message, HTTP status, and optional field details. This package has no YAML configuration or Fx provider. Construct the error where the application detects the failure.

## 1. Choose a public code and message

Use a stable code that callers can recognize. Keep secrets and internal diagnostics out of the public message and details.

The following complete program creates a validation error and shows its JSON representation:

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/errs"
)

func run() error {
    problem := errs.New(
        "INVALID_EMAIL",
        "Email address is invalid",
        http.StatusUnprocessableEntity,
        []errs.Detail{{Field: "email", Message: "must be a valid address"}},
    )
    fmt.Println(problem.Error())
    fmt.Println("HTTP status:", problem.Status)
    return json.NewEncoder(os.Stdout).Encode(problem)
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

`New(code, message, status, details)` returns `*errs.Error`. Supply the status explicitly. The constructor stores your values without validation, normalization, or a default status.

## 2. Check the error's public representation

The program prints this error message:

```text
[INVALID_EMAIL] Email address is invalid
```

Its status is `422`, and its JSON is:

```json
{"code":"INVALID_EMAIL","message":"Email address is invalid","details":[{"field":"email","message":"must be a valid address"}]}
```

`Error()` includes only the code and message. JSON includes `code` and `message`, and omits `details` when the slice is nil or empty. Empty `field` or `message` values within a detail are also omitted. Neither representation includes `Status` or `OriginalError`.

## 3. Return the error to your transport adapter

Inside a use case, return the constructed pointer as an `error`. Your HTTP adapter must apply the status and serialize a response. Creating or JSON-encoding `*errs.Error` does not write an HTTP response, validate the status, or translate the message.

Keep unknown-error handling in the adapter as well. Decide which public response it returns and log private diagnostics separately.

For Fx logger ownership and flushing, follow [logging usage](../logger/usage.md). Pass the structured error to `logger.Error` at the application logging boundary.

## Attach a private original error

When an operation fails because of another error, assign `OriginalError` on a newly constructed error. This is metadata, not Go error-chain wrapping.

This fragment belongs inside a function with `cause error` available. It requires `errs` and `net/http` imports:

```go
problem := errs.New(
    "STORAGE_FAILURE",
    "Please retry later",
    http.StatusServiceUnavailable,
    nil,
)
problem.OriginalError = cause
return problem
```

`*errs.Error` has no `Unwrap` or custom `Is` method. Access `OriginalError` explicitly when logging diagnostics. Assigning it does not make `errors.Is(problem, cause)` succeed.

## Wrap and inspect an application error

Use standard wrapping when a caller needs to recover the structured error through an outer operation error. This complete program demonstrates the distinction between an outer wrapper and `OriginalError`:

```go
package main

import (
    "errors"
    "fmt"
    "net/http"

    "github.com/cristiano-pacheco/bricks/pkg/errs"
)

func main() {
    cause := errors.New("storage unavailable")
    problem := errs.New("STORAGE_FAILURE", "Please retry later", http.StatusServiceUnavailable, nil)
    problem.OriginalError = cause
    wrapped := fmt.Errorf("place order: %w", problem)

    var structured *errs.Error
    fmt.Println(errors.As(wrapped, &structured)) // true
    fmt.Println(errors.Is(wrapped, problem))    // true
    fmt.Println(errors.Is(wrapped, cause))      // false
}
```

`errors.As` finds `*errs.Error` through the outer `%w` wrapper. `errors.Is` compares pointer identity, not error codes, and does not traverse `OriginalError`. Two newly constructed errors with the same code are different pointers. Verify how your downstream adapter handles wrappers before relying on wrapping there.

## Shared errors and mutable fields

`ErrRecordNotFound` is a shared `*errs.Error` with code `RECORD_NOT_FOUND`, message `Record not found`, status `404`, and nil details. Return it unchanged. Create a new error when you need different details or an original cause.

`Error` exports `Code`, `Message`, `Status`, `Details`, and `OriginalError`. `Detail` exports `Field` and `Message`. The constructor retains your details slice rather than cloning it. Treat errors and their slices as immutable once shared, especially across requests. Only `*errs.Error`, not the value type, implements `error`.
