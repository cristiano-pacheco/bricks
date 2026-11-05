# Replace or consume an i18n service

Depend on the public interfaces instead of concrete services. This package declares contracts only. It has no constructors, YAML fields, or lifecycle hooks.

## Inject a translation dependency

1. Add this fragment to a use-case package. It imports `github.com/cristiano-pacheco/bricks/pkg/i18n/ports`:

```go
type ReceiptMessages struct {
    translator ports.TranslationService
}

func NewReceiptMessages(translator ports.TranslationService) *ReceiptMessages {
    return &ReceiptMessages{translator: translator}
}

func (m *ReceiptMessages) Heading() string {
    return m.translator.Translate("checkout", "paid")
}
```

2. Register `NewReceiptMessages` with `fx.Provide` in your application and request the consumer through `fx.Invoke` or `fx.Populate`. The normal `i18n.Module` supplies the interface. For the configuration, filesystem contribution, logger, and application runner, follow the complete inline [translation setup](../usage.md). It keeps the supporting configuration and locale assets separately. Its checkout lookup is expected to print `Pago`.

3. To replace a service, provide the interface explicitly and omit the conflicting standard module provider. This fragment belongs inside an application's `fx.New` arguments and needs `go.uber.org/fx`:

```go
fx.Provide(fx.Annotate(NewApplicationTranslator,
    fx.As(new(ports.TranslationService)))),
```

   `NewApplicationTranslator` is your constructor, not a Bricks function. Its result must implement all three methods below. When replacing just the locale loader or translation layer, assemble the [service constructors](../service/usage.md) yourself rather than installing `i18n.Module` and registering duplicate interface providers. Your replacement owns any cleanup it needs.

## Public contracts

```go
type LocaleLoaderService interface {
    Load(locale string) (map[string]map[string]string, error)
}

type TranslationService interface {
    Translate(domain, key string) string
    TranslateWithData(domain, key string, data map[string]string) string
    GetAllForDomain(domain string) map[string]string
}

type ErrorTranslatorService interface {
    TranslateError(err error) error
}
```

The interfaces do not impose fallback, merge, language normalization, thread-safety, or cleanup policies on custom implementations. Standard implementation behavior belongs to [the service guide](../service/usage.md). Follow its stable-map expectations when supplying a loader to the standard translator.

`i18n.Module` binds the standard error translator to both `ports.ErrorTranslatorService` and `ucdecorator.ErrorTranslator`. These are separate Fx keys even though their method sets match. Providing one custom interface does not automatically provide the other. When using a custom translator with decorators, bind its constructor result to both with two `fx.As` annotations, or explicitly adapt one to the other. Do not install a second decorator translator alongside the standard i18n binding.

For simple translation tasks, loading this interface reference is unnecessary. Use it when replacing services, implementing adapters, or declaring application dependencies.
