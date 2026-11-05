# Construct translation services directly

Translate a message and a structured error without Fx. Use an application Go module with Bricks. Locale loading and translation are synchronous; only the logger needs cleanup.

## Assemble the services

1. Create `cmd/direct-messages/locales/en.json` and `pt-BR.json`:

```json
{"catalog":{"welcome":"Hello {{.Name}}"},"errors":{"QUOTE_QUANTITY":"Quantity must be positive."}}
```

```json
{"catalog":{"welcome":"Olá {{.Name}}"},"errors":{"QUOTE_QUANTITY":"A quantidade deve ser positiva."}}
```

   These are root domain maps, not `dto.Locale` wrappers. English is mandatory as the fallback data source.

2. Create `cmd/direct-messages/main.go`:

```go
package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/config"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/service"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

//go:embed locales/*.json
var files embed.FS

func run() (result error) {
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"i18n.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Sync()) }()
	root, err := fs.Sub(files, "locales")
	if err != nil {
		return err
	}
	loader := service.NewLocaleLoaderService(log, []locale.FileSystem{locale.New(root)})
	translator, err := service.NewTranslationService(loader, config.Config{Language: "pt-BR"}, log)
	if err != nil {
		return err
	}
	fmt.Println(translator.TranslateWithData("catalog", "welcome", map[string]string{"Name": "Ana"}))
	errorTranslator := service.NewErrorTranslatorService(translator)
	original := errs.New("QUOTE_QUANTITY", "Quantity must be positive.", http.StatusBadRequest, nil)
	fmt.Println(errorTranslator.TranslateError(original))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

   Construct the loader first, the translation service next, and the error translator last. `NewTranslationService` returns a loading error, so propagate it. The application defers `Sync` immediately after creating the logger and joins cleanup errors even when later construction fails. Configuration is supplied directly as `config.Config{Language: "pt-BR"}` here. No YAML is read by this program.

3. Run `go run ./cmd/direct-messages`. Expected output:

```text
Olá Ana
[QUOTE_QUANTITY] A quantidade deve ser positiva.
```

   The retained [locale assets](../../../examples/i18n-direct/README.md) contain equivalent JSON files. Create the Go file from step 2. These expected results are source-derived, not an execution record.

## Loader constructor and behavior

`NewLocaleLoaderService(log logger.Logger, fss []locale.FileSystem)` returns `*LocaleLoaderService` without an error or eager loading. Its exported `FileSystems` slice contains the supplied contributions. Configure the slice before use; there is no concurrency or reload protocol for mutating it.

`Load(locale string)` returns `map[string]map[string]string` and an error. It trims the language and converts blank to `en`. It reads `<language>.json` from each supplied root. Read and JSON decoding errors are logged at debug level and skipped, rather than returned individually. At least one successfully decoded file makes loading succeed, including `{}`. Non-English loading with no successful files retries English. No successful English file returns an error, including an empty filesystem slice.

For duplicate domain/key pairs, later files in the explicit constructor slice overwrite earlier ones. Within one JSON object, Go's JSON decoder accepts duplicate keys and later duplicate values can replace or merge earlier ones. Keep duplicate keys out of locale files. Fx filesystem groups have unspecified ordering; their precedence is not deterministic. Missing requested files in some contributions do not trigger per-filesystem English loading. The translation service's separate English map provides key fallback.

## Translation service constructor and methods

`NewTranslationService(loader ports.LocaleLoaderService, cfg config.Config, log logger.Logger)` returns `*TranslationService, error`. It first loads `en`, then the trimmed configured language or `en`. Errors are wrapped with fallback or configured-locale context. It snapshots the maps returned by the loader without a hot-reload or per-request language API. Custom loaders should return maps that remain stable after construction.

- `Translate(domain, key)` returns a configured value, then an English value, then the trimmed key. Blank domain/key returns the trimmed key. Empty-string values count as present.
- `TranslateWithData(domain, key, map[string]string)` uses `text/template` with `missingkey=default`. Empty data skips rendering. Missing variables with nonempty data produce `<no value>`. Template parse or execution errors log warnings and return the original selected string. There is no HTML escaping or error result.
- `GetAllForDomain(domain)` trims the domain and returns a fresh map. It merges English first and configured values second. Blank domains return an empty map. Mutating the returned map does not alter service translations.

Both the loader and translator require a usable logger, including for fallback warnings. Nil dependencies are not a supported configuration. There are no service options, `Close` methods, or YAML fields beyond [the language configuration](../config/usage.md).

## Error translator semantics

`NewErrorTranslatorService(translator ports.TranslationService)` returns `*ErrorTranslatorService`. `TranslateError(err)` uses `errors.As` to find `*errs.Error`. Plain errors and nil remain unchanged. It looks up the error's `Code` under the `errors` domain using `Translate`, not `TranslateWithData`.

A nonempty translation different from the code creates a new `errs.Error` with the same code, status, and details and the translated message. It does not copy `OriginalError`, preserve an outer wrapping error, translate detail messages, interpolate data, or mutate the input. An empty result or a result equal to the code returns the original error unchanged. Do not assume the translated error preserves error identity or cause chains. Avoid typed-nil `*errs.Error` values; they can panic on field access.

To install translation around actual use-case execution, load [the localized-use-case recipe](../../../recipes/localized-usecases.md). Direct `TranslateError` does not install middleware or change every returned application error.
