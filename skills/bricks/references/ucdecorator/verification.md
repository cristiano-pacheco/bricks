# Verify a decorated use case

Read when changing a Bricks decorator's publication, configuration, or effects. Use the application's required test boundary. This guide supplies framework checks, not a replacement test policy.

## 1. Build the selected production publication

Follow [factory setup](factory.md) or [manual chains](manual.md). Use the same raw constructor and public provider as production. Resolve `ucdecorator.UseCase[Input, Output]` for each selected operation. Resolving only the raw pointer does not exercise the chain.

For factory setup, supply configuration, logger, metrics, and translator even when individual flags are false. The loaded YAML does not call `DefaultConfig` to fill omitted flags. Set the intended flags explicitly. Keep environment-changing test graphs sequential.

## 2. Check execution and naming

Invoke the public operation once. Assert the output and required effects. Check exactly one raw execution and one observation per enabled metric. Two operations must coexist under distinct named Input/Output specializations. Aliases do not distinguish generic keys.

`Wrap` infers names from the raw type. For `OrderGetUseCase`, check the selected version's `OrderGetUseCase.Execute` span/log name and `order_get` metric label. Manual chains use their supplied names. Repeated wrapping can change names and duplicate observations.

## 3. Check error translation

Use a known structured error with a locale entry. Assert its stable code and status and the expected translated message. The translator can construct a new error, so pointer identity and an enclosing cause chain need not survive. A raw singleton's `errors.Is` result is not the translated contract. Validation errors keep their own validation contract rather than becoming catalog errors automatically.

For a complete translated setup, use [localized use cases](../../recipes/localized-usecases.md). For filesystem and fallback behavior, load [translation](../i18n/usage.md) only when that contract changes.

## 4. Check disabled behavior

With factory `enabled: false`, `Wrap` returns the raw instance. The same holds when all feature flags are false. Verify that public invocation still works and records no wrapper observations. Disabled flags do not remove factory constructor dependencies.

## 5. Verify only the claimed integrations

- A recording `metrics.UseCaseMetrics` can prove outcome and duration calls. It does not prove Prometheus collection or export. Bricks metrics use global collectors, so repeated construction in one process can collide. Use a controlled dependency or an isolated process rather than resetting unrelated global collectors.
- Tracing needs initialized tracing infrastructure. Check the child span and context propagation through a controlled tracer. Export claims need an actual exporter and collector. Wrappers do not automatically record returned errors or set span status.
- Logging needs an enabled logger and observable output. Translation runs inside metrics and logging, so those layers see the translated return value.
- The chain adds no retries, transactions, authorization, input validation, or panic recovery. Panic unwinding can bypass normal failure metrics, and deferred logging can observe a nil named error. Do not claim panic failure accounting without separate recovery behavior.

For a complete observability setup, select [decorator observability](../../recipes/decorator-observability.md). Report graph validation, execution, output capture, and export as separate results.
