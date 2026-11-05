# Translate application messages

Load Portuguese messages, interpolate a name, and fall back to English for a missing key. This path needs an application Go module with Bricks available and its supported Go toolchain. It does not need the decorator, metrics, or tracing guides.

## Set up translation with Fx

1. Create `cmd/messages/config/base.yaml` with the complete i18n configuration:

```yaml
app:
  i18n:
    language: pt-BR
```

   The module reads `app.i18n`. `language` is an optional string, not a validated language-tag value. The service trims it and uses `en` when empty. It uses the resulting string literally as a filename stem, so `pt_BR` and `pt-BR` are different files. There are no duration or numeric settings.

2. Create `cmd/messages/locales/en.json`:

```json
{
  "catalog": {
    "title": "Products",
    "welcome": "Hello {{.Name}}",
    "only_en": "English fallback"
  },
  "errors": {
    "QUOTE_QUANTITY": "Quantity must be positive."
  }
}
```

   Create `cmd/messages/locales/pt-BR.json`:

```json
{
  "catalog": {
    "title": "Produtos",
    "welcome": "Olá {{.Name}}"
  },
  "errors": {
    "QUOTE_QUANTITY": "A quantidade deve ser positiva."
  }
}
```

   The loader consumes JSON objects containing domain objects and string values. These are locale data, not YAML application configuration. Always provide a valid `en.json`. Translation-service construction loads English even when another language has every requested key.

3. Demonstrate a second module's contribution by creating `cmd/messages/extra/en.json` and `cmd/messages/extra/pt-BR.json` respectively:

```json
{"checkout": {"paid": "Paid"}}
```

```json
{"checkout": {"paid": "Pago"}}
```

   Separate modules can contribute different domains or keys. Keep duplicate domain/key pairs out of Fx filesystem groups, whose order is unspecified.

4. Create `cmd/messages/main.go`:

```go
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/i18n"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"go.uber.org/fx"
)

//go:embed locales/*.json extra/*.json
var files embed.FS

func provideLocales() (locale.FileSystem, error) {
	root, err := fs.Sub(files, "locales")
	return locale.New(root), err
}

func provideExtraLocales() (locale.FileSystem, error) {
	root, err := fs.Sub(files, "extra")
	return locale.New(root), err
}

func newLogger() (logger.Logger, error) {
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"i18n.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	return log, nil
}

func printMessages(translator ports.TranslationService) {
	fmt.Println(translator.Translate("catalog", "title"))
	fmt.Println(translator.TranslateWithData("catalog", "welcome", map[string]string{"Name": "Ana"}))
	fmt.Println(translator.Translate("catalog", "only_en"))
	fmt.Println(translator.Translate("catalog", "unknown"))
	fmt.Println(translator.Translate("checkout", "paid"))
}

func run() (result error) {
	log, err := newLogger()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Sync()) }()
	var translator ports.TranslationService
	app := fx.New(
		i18n.Module,
		fx.Provide(func() logger.Logger { return log },
			fx.Annotate(provideLocales, fx.ResultTags(`group:"locale_filesystems"`)),
			fx.Annotate(provideExtraLocales, fx.ResultTags(`group:"locale_filesystems"`)),
		),
		fx.Populate(&translator),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err == nil {
		printMessages(translator)
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

   `fs.Sub` makes each directory the filesystem root. The loader reads `pt-BR.json`, not `locales/pt-BR.json`. The `locale_filesystems` result group matches the module's input group. An ungrouped filesystem provider does not contribute to loading. `fx.Populate` requests `ports.TranslationService` and therefore constructs its dependencies. The module publishes interfaces, not concrete service pointers. The application constructs the logger before Fx and defers `Sync`, including when graph construction fails. File output allows it to preserve real cleanup errors.

5. Run from your application root:

```sh
APP_CONFIG_DIR=cmd/messages/config APP_ENV=local go run ./cmd/messages
```

   Expected application output, apart from Fx diagnostics:

```text
Produtos
Olá Ana
English fallback
unknown
Pago
```

   The program stops after its lookups. Fallback warnings go to `i18n.log`. Translation services have no shutdown API, and locale data stays in memory. The application flushes its logger after Fx stops. For copying instead of transcribing, use the complete [distributed example directory](../../examples/i18n-translation/README.md). Execution has not been verified for this documentation change.

## Select a fallback scenario

The smallest explicit English configuration is also the full set of supported fields:

```yaml
app:
  i18n:
    language: ""
```

The [minimal file](../../examples/i18n-translation/config/minimal.yaml) produces `Products`, `Hello Ana`, `English fallback`, `unknown`, and `Paid`. The [full file](../../examples/i18n-translation/config/full.yaml) selects Portuguese as above. A missing subtree is a configuration-loading error, so retain `app.i18n`.

Use the distributed [fallback overlay](../../examples/i18n-translation/config/fallback.yaml) with `APP_ENV=fallback` to request `xx`. No `xx.json` exists, so expect English. `base.yaml` remains required. Use the [environment overlay](../../examples/i18n-translation/config/environment.yaml) with `APP_ENV=environment APP_LANGUAGE=pt-BR` to demonstrate an exact `env://APP_LANGUAGE` reference. These are alternative scenarios, not additional i18n fields. See [configuration details](config/usage.md) only for direct loading or configuration-boundary work.

## Lookup behavior

`Translate` trims domain and key, tries the configured locale, then English, then returns the trimmed key. Blank domain or key returns the trimmed key immediately. An existing empty-string translation counts as a match and does not fall back.

`TranslateWithData` uses Go `text/template` and `map[string]string`. Write `{{.Name}}`, not `${Name}`. Empty data returns the untranslated template text. A nonempty map with a missing template variable renders `<no value>`. Parse or execution failure logs a warning and returns the original translated template. There is no returned error and no HTML escaping, so escape for your output context.

`GetAllForDomain` returns a new map containing English keys overwritten by configured keys. Editing that result does not update service state. Language is fixed at construction, not selected from request context or `Accept-Language`. Create separate services for deliberate per-language ownership. Changing config files does not reload an existing service.

## Load only the branch you need

- To build services without Fx or customize loading, read [service constructors](service/usage.md).
- To contribute an embedded or OS filesystem, read [filesystem contributions](locale/usage.md).
- To replace a service with your own implementation, read [public interfaces](ports/usage.md).
- To understand `dto.Locale`, which the loader does not consume, read [the DTO guide](dto/usage.md).
- To translate structured errors around a use case, read [the localized-use-case recipe](../../recipes/localized-usecases.md). Simple message lookup needs none of those integrations.
