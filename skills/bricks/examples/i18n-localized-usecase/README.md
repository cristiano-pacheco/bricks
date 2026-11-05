# Run a localized quote use case

Execute an explicitly wrapped use case with locale-backed structured errors, logging, metrics, and tracing wrappers. The base scenario disables trace export and needs no external service.

1. Copy this directory into an application Go module with Bricks as `cmd/localized-quote`. Retain `main.go`, `config`, and `locales`.
2. Run from the application root:

```sh
APP_CONFIG_DIR=cmd/localized-quote/config APP_ENV=local go run ./cmd/localized-quote 2
```

   Expect `total_cents=250`. Logs go to `i18n.log`. No metrics endpoint is started.

3. Run invalid input:

```sh
APP_CONFIG_DIR=cmd/localized-quote/config APP_ENV=local go run ./cmd/localized-quote 0
```

   Expect `[QUOTE_QUANTITY] A quantidade deve ser positiva.` on stderr and a nonzero exit. The application stops Fx and flushes its logger before reporting the error. `go run` may add an exit-status diagnostic.

4. To disable only error translation, repeat the invalid command with `APP_ENV=untranslated`. Expect `[QUOTE_QUANTITY] Quantity must be positive.` instead. The translator is still provided because the factory requires it regardless of flag values.

## Export spans only when needed

Start a collector accepting OTLP/HTTP at `localhost:4318`, then run with `APP_ENV=export`. The overlay enables plaintext local export with full root sampling. Expect a `QuoteUseCase.Execute` span at the collector after shutdown. See [tracing setup](../../references/otel/trace/usage.md) for a distributed collector configuration and complete exporter fields. The application owns no collector process.

The [localized-use-case recipe](../../recipes/localized-usecases.md) explains all wiring inline, ordering, translation limits, and optional metrics-listener composition. Compilation, YAML loading, Fx validation, runtime operations, and export checks were waived and have not been executed for this change.
