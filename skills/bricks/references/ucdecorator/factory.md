# Configure and explicitly wrap a use case

Use Fx to construct a decorator factory, then provide a wrapped quote use case to its consumer. Installing `ucdecorator.Module` alone does not decorate any application object. You need an application Go module with Bricks. This recommended path enables logging only and requires no external service.

## Create the configured quote command

1. Create `cmd/quote/config/base.yaml`:

```yaml
app:
  ucdecorator:
    enabled: true
    logging: true
```

   Omitted flags are false. The distributed [minimal base.yaml](../../examples/decorator-factory/config/base.yaml) is the same file. The module loads the `app.ucdecorator` subtree using the Bricks configuration loader. `DefaultConfig()` is not applied during YAML loading.

2. Create `cmd/quote/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"go.uber.org/fx"
)

type QuoteInput struct {
	Quantity int
}

type QuoteUseCase struct{}

func (u *QuoteUseCase) Execute(ctx context.Context, input QuoteInput) (int, error) {
	if input.Quantity < 1 {
		return 0, fmt.Errorf("quantity must be positive")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return input.Quantity * 125, nil
}

type quoteTranslator struct{}

func (quoteTranslator) TranslateError(err error) error {
	return fmt.Errorf("quote unavailable: %w", err)
}

func quantity() (int, error) {
	if len(os.Args) < 2 {
		return 2, nil
	}
	return strconv.Atoi(os.Args[1])
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func provideLogger(lifecycle fx.Lifecycle) (logger.Logger, error) {
	cfg := logger.DefaultConfig()
	cfg.Level = "debug"
	cfg.OutputPaths = []string{"decorator.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		return log.Sync()
	}})
	return log, nil
}

func provideQuote(factory *ucdecorator.Factory) ucdecorator.UseCase[QuoteInput, int] {
	return ucdecorator.Wrap[QuoteInput, int](factory, &QuoteUseCase{})
}

func run() error {
	count, err := quantity()
	if err != nil {
		return err
	}
	var quoted ucdecorator.UseCase[QuoteInput, int]
	app := fx.New(
		ucdecorator.Module,
		metrics.Module,
		fx.Provide(
			provideLogger,
			func() ucdecorator.ErrorTranslator { return quoteTranslator{} },
			provideQuote,
		),
		fx.Populate(&quoted),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err == nil {
		var total int
		total, err = quoted.Execute(context.Background(), QuoteInput{Quantity: count})
		if err == nil {
			fmt.Printf("total_cents=%d\n", total)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}
```

   `provideLogger` explicitly returns the `logger.Logger` interface. `metrics.Module` supplies `metrics.UseCaseMetrics`, and the application supplies `ucdecorator.ErrorTranslator`. Fx needs all three even though the minimal configuration disables metrics and translation. `provideQuote` calls `Wrap` and returns the exact generic interface that the application consumes. `fx.Populate` forces construction; the operation itself executes after startup. Shutdown flushes logging before `main` reports an error.

3. Run the command and inspect its file log:

```sh
APP_CONFIG_DIR=cmd/quote/config APP_ENV=local go run ./cmd/quote 2
cat decorator.log
```

   Expect `total_cents=250` and `QuoteUseCase.Execute succeeded` in the log. With input `0`, expect `quantity must be positive`, a failure log, and a nonzero exit after shutdown. It is not prefixed by the translator because translation is disabled.

## Select a configuration scenario

Copy exactly one scenario to `cmd/quote/config/base.yaml`. These are complete alternative decorator configurations, not fragments to merge blindly.

All supported fields, also distributed as [full/base.yaml](../../examples/decorator-factory/scenarios/full/base.yaml):

```yaml
app:
  ucdecorator:
    enabled: true
    debug_mode: false
    logging: true
    metrics: true
    tracing: true
    translation: true
```

This enables every wrapper, not a tracing exporter or metrics endpoint. On input `0`, expect `quote unavailable: quantity must be positive`. See [optional integrations](integrations.md) only when you need external telemetry or locale translation.

Logging only, with every feature flag explicit, also [distributed](../../examples/decorator-factory/scenarios/logging-only/base.yaml):

```yaml
app:
  ucdecorator:
    enabled: true
    debug_mode: false
    logging: true
    metrics: false
    tracing: false
    translation: false
```

Global disablement overrides all other flags, also [distributed](../../examples/decorator-factory/scenarios/disabled/base.yaml):

```yaml
app:
  ucdecorator:
    enabled: false
    debug_mode: true
    logging: true
    metrics: true
    tracing: true
    translation: true
```

`Wrap` returns the original handler. No debug wrappers or wrapper application messages are created. The same Fx providers are still required. A metrics dependency can still register global collectors during construction even when wrapping is disabled. Repeated factory construction is not a way to avoid that dependency's ownership rules.

For debug mode, copy the full example and set `debug_mode: true`, or copy the distributed [debug/base.yaml](../../examples/decorator-factory/scenarios/debug/base.yaml). The provided logger allows debug output. Expect construction messages such as `applying logging decorator` plus `decorator execute start` and corresponding success or error messages during execution. Debug mode is not an independent business wrapper. It surrounds each enabled wrapper and includes use-case name, decorator name, and elapsed whole milliseconds on completion.

## Field reference

All fields are optional booleans. None has units or a nonzero YAML default. The loader requires the selected `app.ucdecorator` subtree to exist. There is no package validation or cross-field error for an enabled chain with every feature disabled.

| Field | Omitted or zero behavior | Meaning when true |
| --- | --- | --- |
| `enabled` | Returns the handler unchanged | Allows the other flags to apply |
| `debug_mode` | No debug wrappers | Adds debug wrappers and application messages for enabled features |
| `logging` | No logging wrapper | Logs success at info or failure at error after execution |
| `metrics` | No metrics wrapper | Observes duration and increments success or error |
| `tracing` | No tracing wrapper | Starts a named span and ends it after execution |
| `translation` | Returns the original error | Calls the translator for non-nil returned errors |

`DefaultConfig()` is a Go helper with `enabled`, logging, metrics, tracing, and translation true, and debug false. It differs from the YAML zero values. There are no YAML name, logger, translator, registry, exporter, or wrapper-order keys. These are Go contracts or separate packages.

## Names, ordering, and direct construction

`Wrap[T, R]` infers `QuoteUseCase.Execute` from the concrete `QuoteUseCase` type. The metric label becomes `quote` after removing `.Execute` and the trailing `UseCase`, then converting to snake case. `HTTPFetchUseCase` becomes `http_fetch`. Anonymous or nil type information falls back to `UseCase.Execute`, with metric label `use_case`. That fallback does not make a nil handler safe to execute. One pointer layer is removed for inference. Explicit names require the separate [manual chain](manual.md), not a factory option.

With every feature enabled, execution enters logging, metrics, tracing, translation, then the base use case. Each enabled debug wrapper surrounds its feature. Translation happens before metrics and logging see the return value. No wrapper records returned errors in the span or sets span status automatically. No wrapper recovers panics. Supply valid non-nil application handlers.

Without Fx, insert this fragment in an error-returning application function with initialized `measurements`, `log`, and `translator`. Import `config` alongside `ucdecorator`. Keep ownership of logger flushing and any tracer shutdown in that function:

```go
cfg, err := config.New[ucdecorator.Config](config.WithPath("app.ucdecorator"))
if err != nil { return err }
factory := ucdecorator.NewFactory(cfg, measurements, log, translator)
quoted := ucdecorator.Wrap[QuoteInput, int](factory, &QuoteUseCase{})
_, err = quoted.Execute(ctx, QuoteInput{Quantity: 2})
return err
```

`NewFactory` takes four arguments, including `config.Config[ucdecorator.Config]`. The older three-argument README call does not match this API. For missing Fx dependencies, inspect the provider's exact return type. A concrete logger or a structurally compatible translator provider is not automatically registered under the required interface. The inline setup makes those interfaces explicit. When publication, flags, or effects change, follow [decorator verification](verification.md). It separates graph checks from actual execution and export.
