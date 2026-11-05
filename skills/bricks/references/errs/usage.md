# Structured errors

Use `github.com/cristiano-pacheco/bricks/pkg/errs` to represent application errors with a public code, message, optional field details, and an HTTP status. The package supplies data and an `error` implementation. It does not write an HTTP response or translate messages.

## Construction and fields

Fragment with `errs` and `net/http` imported:

```go
problem := errs.New(
    "INVALID_EMAIL",
    "Email address is invalid",
    http.StatusUnprocessableEntity,
    []errs.Detail{{Field: "email", Message: "must be a valid address"}},
)
```

`New(code, message, status, details)` returns `*errs.Error`. It stores arguments without validation, normalization, or default status selection. Choose a meaningful code and HTTP status explicitly. `Error()` formats `[INVALID_EMAIL] Email address is invalid`; it does not append details, status, or the original error. Only the pointer type implements `error`.

`Error` exports `Code`, `Message`, `Status`, `Details`, and `OriginalError`. `Detail` exports `Field` and `Message`. The constructor retains the supplied details slice rather than cloning it. Treat returned errors as immutable once shared.

`ErrRecordNotFound` is a shared `*Error` with code `RECORD_NOT_FOUND`, message `Record not found`, status 404, and nil details. Return it unchanged. Construct a fresh error when adding details or an original cause so concurrent requests cannot mutate shared state.

## JSON and HTTP

JSON serialization includes `code` and `message`. Empty or nil `details` is omitted. Within each detail, empty `field` or `message` is omitted. `Status` and `OriginalError` are excluded from JSON. Encoding an error does not apply its status to an HTTP response.

An HTTP adapter must choose the response status, serialization, and treatment of unknown errors. This error object alone does not redact a public message or validate an HTTP status. Put sensitive diagnostic context in private logging rather than `Message` or `Details`.

## Original errors and wrapping

`New` does not take a cause. You may assign `OriginalError` on a freshly constructed error, but it is metadata only. `*errs.Error` has no `Unwrap` or custom `Is` method.

Fragment with `errors`, `fmt`, `errs`, and `net/http` imported:

```go
cause := errors.New("storage unavailable")
problem := errs.New("STORAGE_FAILURE", "Please retry later", http.StatusServiceUnavailable, nil)
problem.OriginalError = cause
wrapped := fmt.Errorf("place order: %w", problem)
var structured *errs.Error
found := errors.As(wrapped, &structured) // true
sameProblem := errors.Is(wrapped, problem) // true, pointer identity
hasCause := errors.Is(wrapped, cause) // false, no Unwrap on problem
```

`errors.As` can recover a structured error through an external `%w` wrapper. `errors.Is` does not compare codes or traverse `OriginalError`. Two separately constructed errors with the same code are distinct pointers. Access `OriginalError` explicitly for diagnostics, or design an application-owned wrapping contract when cause traversal is required. Do not assume all downstream consumers inspect external wrappers in the same way.

There is no Fx module or provider requirement for this package. Construct errors directly inside use cases. For a complete Fx application that consumes configuration and logs a structured error, read [the foundations example](../../examples/foundations/README.md).
