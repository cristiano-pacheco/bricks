# Metrics Package

A Prometheus-based metrics collection package designed for tracking use case execution metrics with Uber FX integration.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

### With Uber FX

To use the module, add it to your fx app:

```go
package main

import (
    "go.uber.org/fx"
    "github.com/cristiano-pacheco/bricks/pkg/metrics"
)

func main() {
    app := fx.New(
        metrics.Module,
        // ... other modules
    )
    app.Run()
}
```

### Standalone

```go
package main

import (
    "time"
    "github.com/cristiano-pacheco/bricks/pkg/metrics"
)

func main() {
    metricsCollector, err := metrics.NewPrometheusUseCaseMetrics()
    if err != nil {
        panic(err)
    }

    // Track use case execution
    start := time.Now()
    
    // ... execute use case ...
    
    metricsCollector.ObserveDuration("user_create", time.Since(start))
    metricsCollector.IncSuccess("user_create")
}
```

## Features

- 📊 **Prometheus Integration**: Native Prometheus metrics with histogram and counter support
- ⏱️ **Duration Tracking**: Histogram-based duration observation with predefined buckets
- ✅ **Success/Error Counting**: Counter-based success and error tracking
- 📦 **FX Integration**: First-class support for Uber FX dependency injection
- 🔧 **Interface-Based Design**: Use the `UseCaseMetrics` interface for easy mocking in tests

## API

### Interface

```go
type UseCaseMetrics interface {
    ObserveDuration(name string, duration time.Duration)
    IncSuccess(name string)
    IncError(name string)
}
```

### Methods

#### `ObserveDuration(name string, duration time.Duration)`

Records the execution duration of a use case in seconds. Uses a histogram with the following buckets:

```go
[]float64{0.005, 0.025, 0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 3, 4, 5}
```

#### `IncSuccess(name string)`

Increments the success counter for the specified use case.

#### `IncError(name string)`

Increments the error counter for the specified use case.

## Prometheus Metrics

The package exposes the following Prometheus metrics:

| Metric Name | Type | Description |
|-------------|------|-------------|
| `usecase_duration_seconds` | Histogram | Duration of use case execution in seconds |
| `usecase_success_total` | Counter | Total successful use case executions |
| `usecase_error_total` | Counter | Total failed use case executions |

All metrics include a `name` label containing the use case name.

## Integration with ucdecorator

This package is designed to work seamlessly with the [`ucdecorator`](../ucdecorator) package, which automatically wraps use cases with metrics collection:

```go
// The ucdecorator package uses UseCaseMetrics internally
decoratedUseCase := ucdecorator.Wrap(factory, myUseCase)
```

## Example: Custom Metrics Collection

```go
package main

import (
    "context"
    "time"
    
    "github.com/cristiano-pacheco/bricks/pkg/metrics"
)

type OrderService struct {
    metrics metrics.UseCaseMetrics
}

func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
    start := time.Now()
    
    // ... process order logic ...
    
    duration := time.Since(start)
    s.metrics.ObserveDuration("process_order", duration)
    
    return nil
}
```

## Testing with Mock

Since the package uses an interface, you can easily create mocks for testing:

```go
package main

import (
    "time"
    "github.com/cristiano-pacheco/bricks/pkg/metrics"
)

// MockUseCaseMetrics implements metrics.UseCaseMetrics for testing
type MockUseCaseMetrics struct {
    Durations []struct {
        Name     string
        Duration time.Duration
    }
    Successes []string
    Errors    []string
}

func (m *MockUseCaseMetrics) ObserveDuration(name string, duration time.Duration) {
    m.Durations = append(m.Durations, struct {
        Name     string
        Duration time.Duration
    }{name, duration})
}

func (m *MockUseCaseMetrics) IncSuccess(name string) {
    m.Successes = append(m.Successes, name)
}

func (m *MockUseCaseMetrics) IncError(name string) {
    m.Errors = append(m.Errors, name)
}
```
