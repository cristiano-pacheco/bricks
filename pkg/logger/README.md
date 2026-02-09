# Logger Package

A robust, production-ready logging package built on top of [Uber Zap](https://github.com/uber-go/zap), providing high-performance structured logging with FX integration.

## Configuration

For a complete configuration example, see [config/config.yaml](config/config.yaml).

## Features

- 🚀 **High Performance**: Built on Zap, one of the fastest structured logging libraries in Go
- 📦 **FX Integration**: First-class support for Uber FX dependency injection
- ⚙️ **Highly Configurable**: Flexible configuration with sensible defaults
- 🎯 **Type-Safe**: Strongly typed fields for structured logging
- 🔧 **Multiple Encodings**: JSON (production) and Console (development) formats
- 🎛️ **Log Sampling**: Built-in support for high-volume log sampling
- 🔌 **Multiple Outputs**: Write logs to stdout, files, or multiple destinations
- 🎨 **Development Mode**: Beautiful colored console output for development

## Installation

```bash
go get go.uber.org/zap
go get go.uber.org/fx
```

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/cristiano-pacheco/bricks/pkg/logger"
)

func main() {
    // Create a logger with default config
    log, err := logger.New(logger.DefaultConfig())
    if err != nil {
        panic(err)
    }
    defer log.Sync()

    // Simple logging
    log.Info("Application started")
    log.Error("Something went wrong", logger.String("error", "connection failed"))
}
```

### Using Options Pattern

```go
log := logger.MustNewWithOptions(
    logger.WithLevel("debug"),
    logger.WithEncoding("console"),
    logger.WithDevelopment(true),
    logger.WithField("service", "my-api"),
    logger.WithField("version", "1.0.0"),
)

log.Debug("Debug message with context")
log.Info("User logged in", 
    logger.String("user_id", "123"),
    logger.String("ip", "192.168.1.1"),
)
```

### FX Integration

```go
package main

import (
    "go.uber.org/fx"
    "github.com/cristiano-pacheco/bricks/pkg/logger"
)

func main() {
    fx.New(
        // Provide logger configuration
        fx.Provide(func() logger.Config {
            return logger.DefaultConfig()
        }),
        
        // Use the logger module with lifecycle management
        logger.ModuleWithLifecycle,
        
        // Use logger in your components
        fx.Invoke(func(log logger.Logger) {
            log.Info("Application initialized")
        }),
    ).Run()
}
```

## Configuration

### Default Configuration (Production)

```go
config := logger.DefaultConfig()
// Level: info
// Encoding: json
// Development: false
// Outputs: stdout
```

### Development Configuration

```go
config := logger.DevelopmentConfig()
// Level: debug
// Encoding: console (colored)
// Development: true
// Outputs: stdout
```

### Custom Configuration

```go
config := logger.Config{
    Level:             "debug",
    Encoding:          "json",
    Development:       false,
    DisableCaller:     false,
    DisableStacktrace: false,
    OutputPaths:       []string{"stdout", "/var/log/app.log"},
    ErrorOutputPaths:  []string{"stderr"},
    InitialFields: map[string]interface{}{
        "service": "my-api",
        "env":     "production",
    },
    SamplingConfig: &logger.SamplingConfig{
        Initial:    100,
        Thereafter: 100,
    },
}
```

## Logging Levels

Available levels (in increasing order of severity):
- `debug` - Detailed debugging information
- `info` - General informational messages
- `warn` - Warning messages
- `error` - Error messages
- `dpanic` - Panic in development, error in production
- `panic` - Panic messages (calls panic())
- `fatal` - Fatal messages (calls os.Exit(1))

## Structured Logging

### Field Types

```go
log.Info("User action",
    logger.String("user_id", "123"),
    logger.Int("age", 30),
    logger.Int64("timestamp", 1234567890),
    logger.Float64("score", 95.5),
    logger.Bool("premium", true),
    logger.Duration("elapsed", time.Second*5),
    logger.Any("metadata", map[string]string{"key": "value"}),
)
```

### Error Logging

```go
err := someOperation()
if err != nil {
    log.Error("Operation failed", logger.Error(err))
    
    // Or use WithError for context
    log.WithError(err).Error("Operation failed",
        logger.String("operation", "save_user"),
    )
}
```

### Context Logger

Create child loggers with additional context:

```go
// Base logger
log := logger.MustNew(logger.DefaultConfig())

// Request-scoped logger
requestLog := log.With(
    logger.String("request_id", "abc-123"),
    logger.String("user_id", "user-456"),
)

// Use throughout request handling
requestLog.Info("Processing request")
requestLog.Info("Request completed")
```

## Best Practices

1. **Use Structured Fields**: Always prefer structured logging over string interpolation
   ```go
   // Good
   log.Info("User registered", logger.String("user_id", userID))
   
   // Avoid
   log.Info(fmt.Sprintf("User %s registered", userID))
   ```

2. **Context Loggers**: Create child loggers for request/operation context
   ```go
   reqLog := log.With(logger.String("request_id", reqID))
   reqLog.Info("Processing")
   ```

3. **Sync on Shutdown**: Always call `Sync()` before application exit
   ```go
   defer log.Sync()
   ```

4. **Use Appropriate Levels**: 
   - `Debug` for development troubleshooting
   - `Info` for general application flow
   - `Warn` for concerning but handled situations
   - `Error` for errors that need attention

## Performance

Zap is designed for high performance:
- Zero allocations for common operations
- Strongly typed fields (no reflection)
- Efficient JSON encoding
- Built-in sampling for high-volume scenarios

## License

Part of the bricks package.
