# Run direct service construction

Translate a Portuguese message and a structured error without Fx. This program configures the language in Go and does not read YAML.

1. Copy this directory into an application Go module with Bricks as `cmd/direct-messages`. Retain `main.go` and both `locales/*.json` files.
2. Run `go run ./cmd/direct-messages` from the application root.
3. Expect:

```text
Olá Ana
[QUOTE_QUANTITY] A quantidade deve ser positiva.
```

The application constructs the logger, locale loader, translation service, and error translator. It propagates construction errors and joins logger-flush errors before reporting failure. It creates `i18n.log`; no external service or shutdown API is needed for translation services.

Use [the service guide](../../references/i18n/service/usage.md) for the complete inline setup and detailed semantics. Compilation and runtime execution were waived by the user and have not been performed for this example.
