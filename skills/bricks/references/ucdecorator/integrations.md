# Add only the integrations your wrapper needs

Choose the integration below after selecting either the [manual chain](manual.md) or [factory setup](factory.md). Those guides own construction and configuration. This reference explains the dependency contracts without requiring the entire observability or i18n documentation set.

## Export spans

1. Start with the factory example and enable `app.ucdecorator.tracing` together with `enabled`.
2. Add the `brickstrace` import and `brickstrace.Module` to its `fx.New` options. Configure `app.open-telemetry` and run an OTLP collector as shown in the self-contained [decorator observability recipe](../../recipes/decorator-observability.md). That recipe distributes the complete combined program and YAML.
3. Call the populated use case after `app.Start` and end work before `app.Stop`. Expect a span named `QuoteUseCase.Execute` in the collector. The tracing module owns provider initialization and shutdown. It must not compete with direct initialization of another process-global provider.

The wrapper passes its new span context to the use case and ends the span after it returns. It does not call `RecordError` or `SetStatus`. An application can record an application error at its source with this optional fragment inside `QuoteUseCase.Execute`, after checking input:

```go
// Add imports for go.opentelemetry.io/otel/trace and go.opentelemetry.io/otel/codes.
if input.Quantity < 1 {
    err := fmt.Errorf("quantity must be positive")
    span := trace.SpanFromContext(ctx)
    span.RecordError(err)
    span.SetStatus(codes.Error, "invalid quantity")
    return 0, err
}
```

This changes the application use case, not the decorator. Use the [tracing guide](../otel/trace/usage.md) only for exporter alternatives, sampling, transport propagation, or direct provider ownership. Without initialization, wrapper span creation does not prove recording or export.

## Measure the wrapped operation

The factory always requires `metrics.UseCaseMetrics`, even when its feature flag is false. The interface is code-only:

```go
type UseCaseMetrics interface {
    ObserveDuration(name string, duration time.Duration)
    IncSuccess(name string)
    IncError(name string)
}
```

Supply `metrics.Module` once per process for Bricks' implementation. It registers global Prometheus collectors during construction and does not remove them at shutdown. Repeated construction can return duplicate-registration errors. The `name` label is explicit for `Chain` and inferred for `Wrap`. The duration includes inner tracing and translation work, not the outer logging completion.

An application may provide its own implementation returning `metrics.UseCaseMetrics` instead of installing `metrics.Module`. This changes collector ownership deliberately; the decorator has no registry configuration. For an endpoint or a custom registry, load [registry wiring](../../recipes/metrics-registries.md). Merely supplying a gatherer or installing a listener does not move the global use-case collectors into it.

## Supply logging with the correct interface

The required logger is `logger.Logger`, not `*slog.Logger`. Provide the interface explicitly as the factory example does, or use `logger.Module` with its actual logger YAML. A constructor returning only `*logger.ZapLogger` needs an explicit interface provider or `fx.Annotate` with `fx.As(new(logger.Logger))`.

Logger shutdown belongs to the provider that creates it. Do not register another flush hook for the same logger merely because several use cases share it. Read the [logger guide](../logger/usage.md) only for backend access, child loggers, or complete logging configuration.

## Translate returned errors

Supply `ucdecorator.ErrorTranslator` with `TranslateError(error) error`. It receives only non-nil errors. The wrapper preserves the use-case output and replaces the returned error with the translator's result. A translator returning nil hides failure from outer metrics and logging. Preserve causes with `%w` or an appropriate structured-error construction when callers need `errors.Is` or HTTP mapping.

The quote examples use an application-provided translator and do not require i18n. To replace it with actual locale loading and structured-error translation, load the [i18n guide](../i18n/usage.md). Its concrete service must be bound to the decorator interface explicitly. Language selection is not a YAML field of `ucdecorator`, and `TranslateError` itself has no request context parameter.
