# Wrap a use case with an explicit chain

Calculate a quote and add logging, measurements, tracing, and error translation around its public `Execute` operation. Use this path when you want explicit names and a fixed chain without factory configuration. You need an application Go module that already depends on Bricks. No external services are needed for this example.

## Create and run the quote command

1. Create `cmd/quote/main.go` with the following program. `QuoteUseCase` validates quantity before doing application work. The translator adds context and preserves the original error with `%w`.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
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

func run() (err error) {
	count, err := quantity()
	if err != nil {
		return err
	}
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"decorator.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, log.Sync()) }()

	measurements, err := metrics.NewPrometheusUseCaseMetrics()
	if err != nil {
		return err
	}
	quoted := ucdecorator.Chain[QuoteInput, int](
		&QuoteUseCase{}, log, measurements, quoteTranslator{},
		"quote", "QuoteUseCase.Execute",
	)
	total, err := quoted.Execute(context.Background(), QuoteInput{Quantity: count})
	if err != nil {
		return err
	}
	fmt.Printf("total_cents=%d\n", total)
	return nil
}
```

   `logger.New` returns a concrete logger that satisfies `logger.Logger`. The file output makes flushing explicit without terminal-specific sync errors. The deferred cleanup joins any flush error with the operation error. The metrics constructor registers process-global collectors once. There is no metrics server here. The tracing wrapper creates a span, but this program does not initialize an exporter.

2. Run from your application root:

```sh
go run ./cmd/quote 2
cat decorator.log
```

   Expect `total_cents=250` on stdout and a `QuoteUseCase.Execute succeeded` log entry in `decorator.log`. Fx is not required for this branch. A copy of the complete [main.go](../../examples/decorator-manual/main.go) travels with the skill.

3. Call the same public operation with invalid input:

```sh
go run ./cmd/quote 0
```

   Expect the command to report `quote unavailable: quantity must be positive` and exit nonzero after flushing. The log contains `QuoteUseCase.Execute failed`. The wrapper returns the use-case output together with the translated error. Invalid command-line text fails before construction rather than going through the decorators.

## Chain behavior and supported alternatives

`Chain[T, R]` accepts the base `UseCase[T, R]`, `logger.Logger`, `metrics.UseCaseMetrics`, `ErrorTranslator`, metric name, and use-case name, in that order. All names are explicit strings. The execution enters logging, metrics, tracing, translation, then the use case. Returning work runs in reverse. Logging therefore receives the translated error; metrics counts the result after translation. A translator that returns nil can turn an application failure into a reported success. Keep non-nil errors non-nil unless that change is intentional.

`UseCase[T, R]` has one method, `Execute(context.Context, T) (R, error)`. `ErrorTranslator` has one method, `TranslateError(error) error`. The translation contract has no context parameter. Request-specific language selection must happen through your own translator arrangement rather than an invented decorator option.

Pass a nil interface for logger, metrics, or translator to omit that wrapper. For example, this fragment replaces the `quoted :=` assignment above for logging and tracing only:

```go
quoted := ucdecorator.Chain[QuoteInput, int](
    &QuoteUseCase{}, log, nil, nil, "quote", "QuoteUseCase.Execute",
)
```

Do not construct metrics if you remove its use from the full program. Tracing always remains in `Chain`; use [factory wrapping](factory.md) when you need independent feature flags. There are no public production `WithLogging` or `WithTracing` constructors. Similarly named helpers in the package's test exports are not consumer APIs. The package does not expose arbitrary ordering through `Chain`.

Manual chains have no YAML contract or functional options. Their names and dependencies are Go arguments. Nil concrete pointers stored inside non-nil interfaces are not the same as nil interfaces. Supply valid dependencies and a non-nil handler. The package does not validate these inputs or recover application panics.

## Load integrations only for the task

For exported spans, custom measurements, or a locale-backed translator, read [optional integrations](integrations.md). For a complete all-feature Fx example, use the [observability recipe](../../recipes/decorator-observability.md). Routine manual-chain use does not require those references. Tracing only starts and ends a span; it does not record returned errors or set error status.
