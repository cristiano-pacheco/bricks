# Run message translation

Load two rooted locale contributions and print configured, interpolated, fallback, and missing-key messages. Use an application Go module with Bricks and its supported Go toolchain.

1. Copy this entire directory into your application as `cmd/messages`. Keep `main.go`, `config`, `locales`, and `extra` together. The embed patterns need both JSON directories when compiling.
2. From the application root, run:

```sh
APP_CONFIG_DIR=cmd/messages/config APP_ENV=local go run ./cmd/messages
```

3. Expect these application output lines, apart from Fx diagnostics:

```text
Produtos
Olá Ana
English fallback
unknown
Pago
```

   `i18n.log` receives fallback warnings. Fx stops before the application flushes its logger. No external service is required.

## Alternative configurations

- `APP_ENV=minimal` selects English through a blank language value.
- `APP_ENV=full` explicitly selects `pt-BR` and contains every i18n field.
- `APP_ENV=fallback` requests unavailable `xx` and selects English through loader fallback.
- `APP_ENV=environment APP_LANGUAGE=pt-BR` demonstrates `env://APP_LANGUAGE` resolution.

Keep `APP_CONFIG_DIR=cmd/messages/config` on each command. The loader always requires `base.yaml` and then merges the named overlay. English output is `Products`, `Hello Ana`, `English fallback`, `unknown`, and `Paid`. Locale JSON is separate from YAML configuration. Changing files does not refresh an already constructed service.

For the numbered setup and API behavior, read [translation usage](../../references/i18n/usage.md). Compilation, configuration loading, Fx assembly, and runtime results were not executed for this change, per the user's waiver.
