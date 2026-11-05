# Return localized errors from a decorated use case

Quote an order with logging, metrics, tracing wrappers, and locale-backed structured-error translation. Invalid quantity returns a Portuguese error while preserving its HTTP status and code. This recipe needs an application Go module with Bricks. The initial configuration disables trace export, so no external service is needed.

## Assemble the application

1. Create `cmd/localized-quote/config/base.yaml`:

```yaml
app:
  i18n:
    language: pt-BR
  ucdecorator:
    enabled: true
    debug_mode: false
    logging: true
    metrics: true
    tracing: true
    translation: true
  open-telemetry:
    app_name: localized-quotes
    trace_enabled: false
```

   The i18n configuration has one field, `language`, an optional string whose blank value means `en`. `pt-BR` selects exactly `pt-BR.json`, not `pt_BR.json`. All decorator booleans are explicit here because YAML loading does not call `DefaultConfig`. They control wrappers created by `Wrap`; they do not provide missing dependencies. `app_name` is required by the tracing module even with export disabled. The logger is configured directly in Go below, so this application has no `app.logger` subtree.

2. Create `cmd/localized-quote/locales/en.json` and `pt-BR.json` respectively:

```json
{"errors":{"QUOTE_QUANTITY":"Quantity must be positive."}}
```

```json
{"errors":{"QUOTE_QUANTITY":"A quantidade deve ser positiva."}}
```

   The domain must be `errors`; its key must exactly match `errs.Error.Code`. Files contain domain objects, not a `code` and `domains` DTO wrapper. English is always needed, even if all Portuguese keys exist.

3. Create `cmd/localized-quote/main.go`:

```go
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/i18n"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"go.uber.org/fx"
)

//go:embed locales/*.json
var files embed.FS

type QuoteInput struct {
	Quantity int
}

type QuoteUseCase struct{}

func (*QuoteUseCase) Execute(ctx context.Context, input QuoteInput) (int, error) {
	if input.Quantity < 1 {
		return 0, errs.New("QUOTE_QUANTITY", "Quantity must be positive.", http.StatusBadRequest, nil)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return input.Quantity * 125, nil
}

func provideLocales() (locale.FileSystem, error) {
	root, err := fs.Sub(files, "locales")
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

func provideQuote(factory *ucdecorator.Factory) ucdecorator.UseCase[QuoteInput, int] {
	return ucdecorator.Wrap[QuoteInput, int](factory, &QuoteUseCase{})
}

func run() (result error) {
	log, err := newLogger()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Sync()) }()
	count := 2
	if len(os.Args) > 1 {
		var err error
		count, err = strconv.Atoi(os.Args[1])
		if err != nil {
			return err
		}
	}
	var quote ucdecorator.UseCase[QuoteInput, int]
	app := fx.New(
		brickstrace.Module,
		i18n.Module,
		metrics.Module,
		ucdecorator.Module,
		fx.Provide(func() logger.Logger { return log }, provideQuote,
			fx.Annotate(provideLocales, fx.ResultTags(`group:"locale_filesystems"`)),
		),
		fx.Populate(&quote),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err == nil {
		var total int
		total, err = quote.Execute(context.Background(), QuoteInput{Quantity: count})
		if err == nil {
			fmt.Printf("total_cents=%d\n", total)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
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

   Validation belongs at the start of `QuoteUseCase.Execute`. The explicit `provideQuote` call wraps the use case; installing modules alone does not decorate it. `i18n.Module` provides both the translation-service interface and the decorator's `ErrorTranslator`, including the required raw configuration bridge. The grouped, rooted filesystem provider supplies its locale data. `metrics.Module` supplies the factory's metrics interface. The application constructs and owns the logger and joins its flush error after stopping Fx, even if another constructor fails.

4. Run a valid operation from the application root:

```sh
APP_CONFIG_DIR=cmd/localized-quote/config APP_ENV=local go run ./cmd/localized-quote 2
```

   Expect `total_cents=250` and a use-case log in `i18n.log`. Now run invalid input:

```sh
APP_CONFIG_DIR=cmd/localized-quote/config APP_ENV=local go run ./cmd/localized-quote 0
```

   Expect `[QUOTE_QUANTITY] A quantidade deve ser positiva.` on stderr and a nonzero exit after Fx shutdown and logger flushing. `go run` can also print its exit-status diagnostic. The returned structured error keeps code `QUOTE_QUANTITY` and status 400. This example does not start an HTTP server or expose a metrics endpoint.

   Copy the retained [configuration and locale assets](../examples/i18n-localized-usecase/README.md) if preferred, and create the Go file from step 3. The locale assets include additional message keys, but the `errors` entries above are identical. Results here are expected behavior from source inspection, not recorded runtime proof.

## What the chain does

With all wrapper flags true, entry order is logging, metrics, tracing, translation, then the use case. Translation processes the error first on return, so the outer wrappers see the translated result. Factory naming derives `QuoteUseCase.Execute` and metrics name `quote`. Translation does not create request-scoped language selection. One translator has one language for its lifetime.

Plain errors such as context cancellation remain unchanged. A missing or empty error-code translation also leaves the original error unchanged. Only the top-level message of an `errs.Error` changes. Detail messages, successful output, arbitrary nested strings, and validation errors that are not `errs.Error` are not automatically translated. Error translation does not interpolate templates. It creates a new structured error and discards `OriginalError` and outer wrapping context when translation succeeds. Preserve diagnostic causes separately when your application requires them.

Metrics use the package's process-global collectors; installing a listener or a custom registry elsewhere does not unify ownership. Run this example as its own process rather than constructing duplicate metrics modules repeatedly in one process. This initial recipe initializes a non-recording tracer and does not export spans.

## Select an optional scenario

To retain observability but return English messages unchanged, use [untranslated.yaml](../examples/i18n-localized-usecase/config/untranslated.yaml) as the `APP_ENV=untranslated` overlay. It sets `app.ucdecorator.translation: false`; the factory still requires its translator dependency. Invalid quantity then returns `[QUOTE_QUANTITY] Quantity must be positive.`. Global `enabled: false` would return the original use case with no wrappers. See [factory configuration](../references/ucdecorator/factory.md) only for detailed flags, naming, or debug wrappers.

For actual trace export, start an OTLP/HTTP receiver and use [export.yaml](../examples/i18n-localized-usecase/config/export.yaml) with `APP_ENV=export`:

```yaml
app:
  open-telemetry:
    app_name: localized-quotes
    trace_enabled: true
    trace_url: localhost:4318
    exporter_type: http
    insecure: true
    sample_rate: 1.0
```

The base configuration still supplies i18n and decorators. Expect a `QuoteUseCase.Execute` span at the receiver after the operation and shutdown. Use [tracing setup](../references/otel/trace/usage.md) for the receiver configuration, complete exporter fields, and transport alternatives. Fx owns tracing startup and shutdown; work runs after `app.Start`, not during constructor execution.

For scraping the use-case measurements, load [registry-to-listener wiring](metrics-registries.md). For complete i18n loading or template semantics, load [translation usage](../references/i18n/usage.md). For custom service implementations, load [the ports guide](../references/i18n/ports/usage.md). None of those optional references is required to reproduce this localized-error path.
