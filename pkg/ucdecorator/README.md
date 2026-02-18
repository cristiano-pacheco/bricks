# Use Case Decorator Package

A decorator pattern implementation for use cases that provides automatic logging, metrics collection, tracing, and error translation with Uber FX integration.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks/pkg/ucdecorator
```

## Usage

### With Uber FX

Add the module to your fx app:

```go
package main

import (
    "go.uber.org/fx"
    "github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
)

func main() {
    app := fx.New(
        ucdecorator.Module,
        // ... other modules
    )
    app.Run()
}
```

## Features

- 🔗 **Decorator Chain**: Composable decorators that execute in a specific order
- 📊 **Metrics Integration**: Automatic duration, success, and error tracking via Prometheus
- 📝 **Logging**: Error logging for failed use case executions
- 🔍 **Tracing**: OpenTelemetry span creation for distributed tracing
- 🌐 **Error Translation**: Automatic error translation for localization
- 📦 **FX Integration**: First-class support for Uber FX dependency injection
- 🏭 **Factory Pattern**: Automatic use case name inference for metrics

## Decorator Execution Order

Decorators are applied in the following order (inside-out):

```
Logging → Metrics → Tracing → Translation → Base Use Case
```

This means:
1. **Translation** wraps the base use case (translates errors on the way out)
2. **Tracing** wraps translation (creates span for the entire operation)
3. **Metrics** wraps tracing (records duration and success/error counts)
4. **Logging** wraps metrics (logs errors after all other decorators complete)

## API

### Interfaces

#### `UseCase[T any, R any]`

The base interface that all use cases implement:

```go
type UseCase[T any, R any] interface {
    Execute(ctx context.Context, input T) (R, error)
}
```

#### `ErrorTranslator`

Interface for translating errors to localized versions:

```go
type ErrorTranslator interface {
    TranslateError(err error) error
}
```

### Functions

#### `Wrap[T any, R any](factory *Factory, handler UseCase[T, R]) UseCase[T, R]`

Wraps a use case with all decorators using the factory:

```go
decoratedUseCase := ucdecorator.Wrap(factory, myUseCase)
```

#### `Chain[T any, R any](handler UseCase[T, R], log logger.Logger, useCaseMetrics metrics.UseCaseMetrics, translator ErrorTranslator, metricName string, useCaseName string) UseCase[T, R]`

Composes all decorators in the expected execution order:

```go
decorated := ucdecorator.Chain(
    baseUseCase,
    logger,
    metrics,
    translator,
    "user_create",
    "UserCreateUseCase.Execute",
)
```

### Factory

The `Factory` automatically infers use case names from the concrete type:

```go
factory := ucdecorator.NewFactory(useCaseMetrics, logger, translator)
```

## Complete FX Integration Example

### Step 1: Define Your Use Cases

```go
// internal/modules/catalog/usecase/category_list.go
package usecase

type CategoryListInput struct {
    Page    int
    PerPage int
}

type CategoryListOutput struct {
    Categories []Category
    Total      int64
}

type CategoryListUseCase struct {
    repo ports.CategoryRepository
}

func NewCategoryListUseCase(repo ports.CategoryRepository) *CategoryListUseCase {
    return &CategoryListUseCase{repo: repo}
}

func (uc *CategoryListUseCase) Execute(ctx context.Context, input CategoryListInput) (CategoryListOutput, error) {
    categories, total, err := uc.repo.List(ctx, input.Page, input.PerPage)
    if err != nil {
        return CategoryListOutput{}, err
    }
    return CategoryListOutput{Categories: categories, Total: total}, nil
}
```

### Step 2: Define Decorator Provider Function

Create a provider function that uses `fx.In` and `fx.Out` to decorate all use cases:

```go
// internal/modules/catalog/module.go
package catalog

import (
    "go.uber.org/fx"
    
    "github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
    "your-app/internal/modules/catalog/usecase"
)

type decorateIn struct {
    fx.In

    Factory *ucdecorator.Factory

    CategoryCreate *usecase.CategoryCreateUseCase
    CategoryList   *usecase.CategoryListUseCase
}

type decorateOut struct {
    fx.Out

    CategoryCreate ucdecorator.UseCase[usecase.CategoryCreateInput, usecase.CategoryCreateOutput]
    CategoryList   ucdecorator.UseCase[usecase.CategoryListInput, usecase.CategoryListOutput]
}

func provideDecoratedUseCases(in decorateIn) decorateOut {
    return decorateOut{
        CategoryCreate: ucdecorator.Wrap(in.Factory, in.CategoryCreate),
        CategoryList:   ucdecorator.Wrap(in.Factory, in.CategoryList),
    }
}
```

### Step 3: Define the FX Module

```go
var Module = fx.Module(
    "catalog",
    fx.Provide(
        // Use Cases (raw, undecorated)
        usecase.NewCategoryCreateUseCase,
        usecase.NewCategoryListUseCase,

        // Decorated Use Cases
        provideDecoratedUseCases,

        // Handlers and Routers
        handler.NewCategoryHandler,
    ),
)
```

### Step 4: Inject Decorated Use Cases in Handlers

```go
// internal/modules/catalog/http/chi/handler/category_handler.go
package handler

import (
    "github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
    "your-app/internal/modules/catalog/usecase"
)

type CategoryHandler struct {
    categoryListUseCase   ucdecorator.UseCase[usecase.CategoryListInput, usecase.CategoryListOutput]
    categoryCreateUseCase ucdecorator.UseCase[usecase.CategoryCreateInput, usecase.CategoryCreateOutput]
    errorHandler          response.ErrorHandler
    logger                logger.Logger
}

func NewCategoryHandler(
    categoryListUseCase ucdecorator.UseCase[usecase.CategoryListInput, usecase.CategoryListOutput],
    categoryCreateUseCase ucdecorator.UseCase[usecase.CategoryCreateInput, usecase.CategoryCreateOutput],
    errorHandler response.ErrorHandler,
    logger logger.Logger,
) *CategoryHandler {
    return &CategoryHandler{
        categoryListUseCase:   categoryListUseCase,
        categoryCreateUseCase: categoryCreateUseCase,
        errorHandler:          errorHandler,
        logger:                logger,
    }
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
    input := usecase.CategoryListInput{
        Page:    1,
        PerPage: 20,
    }
    
    output, err := h.categoryListUseCase.Execute(r.Context(), input)
    if err != nil {
        h.errorHandler.Handle(w, err)
        return
    }
    
    // ... render response
}
```

## Metric Name Inference

The factory automatically infers metric names from the use case type name:

| Use Case Type | Metric Name |
|---------------|-------------|
| `UserCreateUseCase` | `user_create` |
| `ProductListUseCase` | `product_list` |
| `CategoryGetUseCase` | `category_get` |

The name is converted to snake_case and the `UseCase` suffix is removed.

## Dependencies

This package depends on:

- [`pkg/logger`](../logger) - For error logging
- [`pkg/metrics`](../metrics) - For Prometheus metrics collection
- [`pkg/otel/trace`](../otel/trace) - For OpenTelemetry tracing

Make sure these modules are also included in your FX application.
