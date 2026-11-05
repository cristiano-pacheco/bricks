# Load i18n configuration

Read the language selected for translation. This package owns one field, not locale messages or a service lifecycle. Use an application Go module with Bricks available.

## Read the configuration

1. Create `cmd/language/config/base.yaml`. This is the minimal explicit configuration:

```yaml
app:
  i18n:
    language: ""
```

   For a full-field Portuguese configuration, replace it with:

```yaml
app:
  i18n:
    language: pt-BR
```

   Both files contain every supported field. Matching [minimal.yaml](../../../examples/i18n-translation/config/minimal.yaml) and [full.yaml](../../../examples/i18n-translation/config/full.yaml) travel with the skill.

2. Create `cmd/language/main.go`:

```go
package main

import (
    "fmt"
    "os"

    bricksconfig "github.com/cristiano-pacheco/bricks/pkg/config"
    i18nconfig "github.com/cristiano-pacheco/bricks/pkg/i18n/config"
)

func run() error {
    value, err := bricksconfig.New[i18nconfig.Config](bricksconfig.WithPath("app.i18n"))
    if err != nil { return err }
    fmt.Printf("configured_language=%q\n", value.Get().Language)
    return nil
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

3. Run `APP_CONFIG_DIR=cmd/language/config APP_ENV=local go run ./cmd/language`. Expect `configured_language="pt-BR"` for the full configuration. The minimal file prints an empty string. The translation service, not the configuration loader, converts blank or whitespace-only language to `en`. No cleanup is needed for this value-only operation.

## Field contract

`Config.Language` is a string tagged `config:"language"`. It is optional and has a Go zero value of `""`. There are no units, enum validation, or locale-tag normalization. The loader uses the trimmed service language literally as `<language>.json`; provide `pt-BR.json` for `pt-BR`. Treat configuration as trusted, not arbitrary user input. A missing `app.i18n` subtree fails `WithPath` loading. There are no fallback-language, locale-directory, request-language, or reload YAML fields.

`i18n.Module` provides `config.Config[i18nconfig.Config]`, then unwraps it with `Get()` to provide the raw `i18nconfig.Config` required by the service constructor. Do not add a second raw provider when using that module.

The generic loader reads required `base.yaml` plus optional `<APP_ENV>.yaml` under `APP_CONFIG_DIR`, whose defaults are `config` and `local`. A distributed [fallback overlay](../../../examples/i18n-translation/config/fallback.yaml) selects `xx`; the [environment overlay](../../../examples/i18n-translation/config/environment.yaml) uses the exact value `env://APP_LANGUAGE`. The base file remains required. An environment-variable reference is loader behavior, not a separate i18n option.

For actual JSON loading and lookups, use [the translation setup](../usage.md). The configuration example only proves which value you loaded when you run it, not whether locale files exist.
