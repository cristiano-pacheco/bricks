# i18n Package

An internationalization package that provides locale loading, translation lookups, and error translation with Uber FX integration.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

### With Uber FX

Add the module to your fx app along with a `locale.FileSystem` provider:

```go
package main

import (
    "embed"

    "go.uber.org/fx"
    "github.com/cristiano-pacheco/bricks/pkg/i18n"
    "github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
)

//go:embed locales
var localesFS embed.FS

func main() {
    app := fx.New(
        i18n.Module,
        fx.Provide(func() locale.FileSystem {
            return locale.New(localesFS)
        }),
        // ... other modules
    )
    app.Run()
}
```

### Configuration

The module reads its configuration from the `app.i18n` key:

```yaml
app:
  i18n:
    language: "pt-BR"
```

| Field      | Type   | Description                                          |
|------------|--------|------------------------------------------------------|
| `language` | string | BCP 47 locale code (e.g. `en`, `pt-BR`). Defaults to `en` if empty. |

## Locale File Format

Locale files are JSON files named after the locale code and placed inside the embedded filesystem. Each file maps **domain** names to key/value translation pairs.

**Directory layout:**

```
locales/
  en.json
  pt-BR.json
```

**File format (`en.json`):**

```json
{
  "errors": {
    "user.not_found": "User not found.",
    "user.already_exists": "User already exists."
  },
  "validation": {
    "required": "The field {{.field}} is required.",
    "max_length": "The field {{.field}} must not exceed {{.max}} characters."
  }
}
```

Top-level keys are **domains** and inner keys are **translation keys**. Values support Go `text/template` syntax for interpolation.

## Features

- **Locale Loading**: Reads JSON locale files from any `fs.FS` (embed, OS, memory)
- **Automatic Fallback**: Falls back to `en` locale when the configured locale is missing or a key is not found
- **Template Interpolation**: Supports `text/template` placeholders in translation values
- **Error Translation**: Translates typed `brickserrs.Error` values using the `errors` domain
- **FX Integration**: First-class support for Uber FX dependency injection
- **Interface-Based Design**: All services are backed by interfaces for easy mocking in tests

## API

### Interfaces

#### `ports.LocaleLoaderService`

Loads locale data from the filesystem for a given locale code:

```go
type LocaleLoaderService interface {
    Load(locale string) (map[string]map[string]string, error)
}
```

#### `ports.TranslationService`

Provides translation lookups against the loaded locale data:

```go
type TranslationService interface {
    Translate(domain, key string) string
    TranslateWithData(domain, key string, data map[string]string) string
    GetAllForDomain(domain string) map[string]string
}
```

#### `ports.ErrorTranslatorService`

Translates typed `brickserrs.Error` values into localized messages:

```go
type ErrorTranslatorService interface {
    TranslateError(err error) error
}
```

### Methods

#### `TranslationService.Translate(domain, key string) string`

Returns the translation for the given domain and key. Falls back to the `en` locale if the key is absent in the configured locale, and returns the key itself if not found anywhere.

#### `TranslationService.TranslateWithData(domain, key string, data map[string]string) string`

Same as `Translate`, but renders the result as a Go template with `data` as the template context.

```go
// Locale value: "The field {{.field}} is required."
msg := translationService.TranslateWithData("validation", "required", map[string]string{
    "field": "email",
})
// → "The field email is required."
```

#### `TranslationService.GetAllForDomain(domain string) map[string]string`

Returns all key/value pairs for the given domain, merging the fallback (`en`) and configured locale (configured locale wins on conflicts).

#### `ErrorTranslatorService.TranslateError(err error) error`

Unwraps a `*brickserrs.Error` and looks up its `Code` in the `errors` domain. Returns a new error with the translated message, or the original error unchanged if the code has no translation.

### `locale.FileSystem`

A thin wrapper around `fs.FS` used to provide the locale files to the loader:

```go
//go:embed locales
var localesFS embed.FS

fs := locale.New(localesFS)
```

## Fallback Strategy

The package applies a two-level fallback:

1. **Locale-level fallback** (`LocaleLoaderService.Load`): if the requested locale file cannot be read, it falls back to `en.json` and logs a warning.
2. **Key-level fallback** (`TranslationService.Translate`): if a key is missing in the configured locale, it tries the `en` translations and logs a warning. If still not found, it returns the raw key.

## Complete FX Integration Example

### Step 1: Embed Your Locale Files

```go
// internal/locales/locales.go
package locales

import "embed"

//go:embed *.json
var FS embed.FS
```

**`internal/locales/en.json`:**

```json
{
  "errors": {
    "user.not_found": "User not found.",
    "order.invalid_status": "Invalid order status."
  }
}
```

**`internal/locales/pt-BR.json`:**

```json
{
  "errors": {
    "user.not_found": "Usuário não encontrado.",
    "order.invalid_status": "Status do pedido inválido."
  }
}
```

### Step 2: Provide the FileSystem and Register the Module

```go
// internal/app/app.go
package app

import (
    "go.uber.org/fx"
    "github.com/cristiano-pacheco/bricks/pkg/i18n"
    "github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
    "your-app/internal/locales"
)

var Module = fx.New(
    i18n.Module,
    fx.Provide(func() locale.FileSystem {
        return locale.New(locales.FS)
    }),
)
```

### Step 3: Inject `TranslationService` in a Use Case

```go
// internal/modules/order/usecase/order_create.go
package usecase

import (
    "context"

    "github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
)

type OrderCreateUseCase struct {
    translator ports.TranslationService
}

func NewOrderCreateUseCase(translator ports.TranslationService) *OrderCreateUseCase {
    return &OrderCreateUseCase{translator: translator}
}

func (uc *OrderCreateUseCase) Execute(ctx context.Context, input OrderCreateInput) (OrderCreateOutput, error) {
    // Translate a validation message with interpolated data
    msg := uc.translator.TranslateWithData("validation", "required", map[string]string{
        "field": "customer_id",
    })
    _ = msg
    // ...
    return OrderCreateOutput{}, nil
}
```

### Step 4: Automatic Error Translation via `ucdecorator`

When `i18n.Module` is combined with `ucdecorator.Module`, the `ErrorTranslatorService` is automatically wired as the `ucdecorator.ErrorTranslator`. Any `*brickserrs.Error` returned by a decorated use case will have its message translated before reaching the caller.

```go
var Module = fx.New(
    i18n.Module,
    ucdecorator.Module,
    // ...
)
```

## Testing with Mocks

Since all services are interface-backed, you can substitute them in tests:

```go
type MockTranslationService struct{}

func (m *MockTranslationService) Translate(domain, key string) string {
    return key
}

func (m *MockTranslationService) TranslateWithData(domain, key string, data map[string]string) string {
    return key
}

func (m *MockTranslationService) GetAllForDomain(domain string) map[string]string {
    return map[string]string{}
}
```

## Dependencies

This package depends on:

- [`pkg/config`](../config) - For reading `app.i18n` configuration
- [`pkg/logger`](../logger) - For logging fallback warnings
- [`pkg/errs`](../errs) - For typed error unwrapping in `ErrorTranslatorService`
- [`pkg/ucdecorator`](../ucdecorator) - The `ErrorTranslator` interface is satisfied by `ErrorTranslatorService`

Make sure these modules are also included in your FX application.
