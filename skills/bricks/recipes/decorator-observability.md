# Observe and translate one wrapped quote operation

Run a configured quote use case with logging, Prometheus measurements, tracing, and application-provided error translation. The operation runs after Fx startup and before shutdown. You need an application Go module with Bricks and an OTLP/HTTP collector for span delivery. There is no metrics listener in this command.

## Set up the complete composition

1. Copy the distributed [collector.yaml](../examples/tracing-span/collector.yaml) into your application root and run the collector in a separate terminal:

```sh
docker run --rm --name bricks-quote-collector \
  -p 127.0.0.1:4318:4318 \
  -v "$PWD/collector.yaml:/etc/otelcol/config.yaml:ro" \
  otel/opentelemetry-collector:0.123.0 --config=/etc/otelcol/config.yaml
```

   This command can download an image. The collector is an external dependency. Its debug output is the place to observe delivered spans, not evidence supplied by successful Fx startup.

2. Create `cmd/quote/config/base.yaml`:

```yaml
app:
  ucdecorator:
    enabled: true
    debug_mode: true
    logging: true
    metrics: true
    tracing: true
    translation: true
  open-telemetry:
    app_name: quotes
    trace_enabled: true
    trace_url: localhost:4318
    exporter_type: http
    insecure: true
    sample_rate: 1.0
```

   The decorator flags choose wrappers. The separate tracing configuration chooses a real exporter. Plaintext and full root sampling are local demonstration settings, not a production policy. A matching [base.yaml](../examples/decorator-observability/config/base.yaml) travels with this example.

3. Create `cmd/quote/main.go`. This program provides every dependency, explicitly calls `Wrap`, and executes the resulting interface. The translator adds application context while preserving the cause.

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
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
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
		brickstrace.Module,
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

   The logger owns a file and registers its flush hook. Metrics owns process-global collectors. Tracing owns a process-global provider and flushes during shutdown. The application has no extra cleanup for the factory itself. `errors.Join` preserves the execution error and stop errors. `QuoteUseCase` validates before calculating its result, rather than using a startup hook as the application operation.

4. Run the operation from your application root:

```sh
APP_CONFIG_DIR=cmd/quote/config APP_ENV=local go run ./cmd/quote 2
cat decorator.log
```

   Expect `total_cents=250`, wrapper application and execution messages in the file log, and a collector span named `QuoteUseCase.Execute` with `service.name` equal to `quotes`. The process exits after shutdown. The metrics implementation records a `usecase_success_total` sample with `name="quote"` and one duration observation in the default registry. This short-lived command does not expose them over HTTP. Use the [registry-to-listener recipe](metrics-registries.md) only when your application needs a scraping endpoint.

5. Run with quantity `0`:

```sh
APP_CONFIG_DIR=cmd/quote/config APP_ENV=local go run ./cmd/quote 0
```

   Expect `quote unavailable: quantity must be positive`, a failure log, and nonzero exit after cleanup. The error counter replaces the success increment for this operation. The span still ends, but the decorator does not automatically record the returned error or set an error status. Translation happens before outer metrics and logging see the result. Debug wrappers include each enabled feature, so the outer debug messages also see the translated error.

   Copy the distributed [main.go](../examples/decorator-observability/main.go) rather than transcribing the program if preferred. Stop the collector with Ctrl-C when finished. Expected output here describes the source-backed contract, not a recorded execution result.

## Change one part deliberately

For flags, zero values, global disablement, or inferred naming, load [factory configuration](../references/ucdecorator/factory.md). For explicit names without Fx configuration, load the [manual-chain guide](../references/ucdecorator/manual.md). Both paths enter logging, metrics, tracing, translation, then the use case when all features are present.

The concrete translator is deliberately an application type. For locale-backed structured errors, load the [i18n guide](../references/i18n/usage.md) and bind its translator under `ucdecorator.ErrorTranslator`. For field-by-field exporter settings and alternate transports, load the [tracing guide](../references/otel/trace/usage.md). For adding error status at the application source, read [optional decorator integrations](../references/ucdecorator/integrations.md).
