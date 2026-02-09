# Redis Package

A robust and highly configurable Redis client package for Go, built on top of [go-redis/redis](https://github.com/redis/go-redis), with support for single-node, cluster, sentinel, and failover setups.

## Configuration

For a complete configuration example, see [config/config.yaml](config/config.yaml).

## Features

- 🔌 **Multiple Client Types**: Single-node, Cluster, Sentinel, and Failover support
- ⚙️ **Highly Configurable**: Extensive configuration options for connection pools, timeouts, and retries
- 🔄 **Automatic Retries**: Built-in retry mechanism with exponential backoff
- 🔒 **TLS Support**: Secure connections with TLS/SSL
- 📊 **Metrics & Statistics**: Built-in metrics collection and pool statistics
- 🎯 **Namespace Support**: Key namespacing for multi-tenant applications
- 🔌 **Uber FX Integration**: First-class support for Uber FX dependency injection
- 🎨 **Functional Options**: Flexible configuration using the functional options pattern
- 🛡️ **Type-Safe Errors**: Custom error types for better error handling

## Installation

```bash
go get github.com/cristiano-pacheco/bricks/pkg/redis
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/cristiano-pacheco/bricks/pkg/redis"
)

func main() {
    // Create a configuration
    cfg := redis.Config{
        URL:  "redis://localhost:6379",
        Type: redis.ClientTypeSingleNode,
        DB:   0,
    }

    // Create a new client
    ctx := context.Background()
    client, err := redis.NewClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Use the underlying Redis client
    err = client.UniversalClient().Set(ctx, "key", "value", time.Hour).Err()
    if err != nil {
        panic(err)
    }

    val, err := client.UniversalClient().Get(ctx, "key").Result()
    if err != nil {
        panic(err)
    }
    fmt.Println("key:", val)
}
```

### With Uber FX

```go
package main

import (
    "context"

    "github.com/cristiano-pacheco/bricks/pkg/redis"
    "go.uber.org/fx"
)

func main() {
    app := fx.New(
        fx.Provide(
            func() redis.Config {
                return redis.Config{
                    URL:  "redis://localhost:6379",
                    Type: redis.ClientTypeSingleNode,
                }
            },
        ),
        redis.Module,
        fx.Invoke(func(client *redis.Client) {
            // Use the client
        }),
    )

    app.Run()
}
```

## Configuration

### Client Types

The package supports four client types:

```go
const (
    ClientTypeSingleNode ClientType = "single_node" // Single Redis instance
    ClientTypeCluster    ClientType = "cluster"     // Redis Cluster
    ClientTypeSentinel   ClientType = "sentinel"    // Redis Sentinel
    ClientTypeFailover   ClientType = "failover"    // Redis Failover
)
```

### Configuration Options

```go
type Config struct {
    // Connection parameters
    URL      string     // Redis URL (e.g., redis://localhost:6379)
    Password string     // Redis password
    DB       int        // Database number (0-15 for single node)
    Type     ClientType // Client type

    // Cluster-specific
    ClusterAddrs []string // Cluster node addresses

    // Sentinel-specific
    SentinelAddrs    []string // Sentinel addresses
    MasterName       string   // Master name
    SentinelPassword string   // Sentinel password
    SentinelUsername string   // Sentinel username

    // Connection pool settings
    PoolSize           int           // Maximum number of socket connections
    MinIdleConns       int           // Minimum number of idle connections
    MaxIdleConns       int           // Maximum number of idle connections
    ConnMaxIdleTime    time.Duration // Amount of time after which client closes idle connections
    ConnMaxLifetime    time.Duration // Connection age at which client retires the connection
    
    // Timeout settings
    DialTimeout    time.Duration // Dial timeout for establishing new connections
    ReadTimeout    time.Duration // Timeout for socket reads
    WriteTimeout   time.Duration // Timeout for socket writes
    CommandTimeout time.Duration // Default timeout for commands

    // Retry settings
    MaxRetries      int           // Maximum number of retries
    MinRetryBackoff time.Duration // Minimum backoff between retries
    MaxRetryBackoff time.Duration // Maximum backoff between retries

    // TLS configuration
    EnableTLS     bool   // Enable TLS
    TLSSkipVerify bool   // Skip TLS certificate verification
    TLSServerName string // TLS server name

    // Advanced options
    Namespace     string // Key namespace prefix
    EnableMetrics bool   // Enable metrics collection
}
```

### Default Values

The package automatically sets sensible defaults:

```go
DialTimeout:       5 * time.Second
ReadTimeout:       3 * time.Second
WriteTimeout:      3 * time.Second
PoolSize:          10
MinIdleConns:      2
MaxRetries:        3
MinRetryBackoff:   8 * time.Millisecond
MaxRetryBackoff:   512 * time.Millisecond
PoolTimeout:       4 * time.Second
ConnMaxIdleTime:   30 * time.Minute
ConnMaxLifetime:   1 * time.Hour
CommandTimeout:    5 * time.Second
```

## Usage Examples

### Single-Node Redis

```go
cfg := redis.Config{
    URL:      "redis://localhost:6379",
    Type:     redis.ClientTypeSingleNode,
    Password: "my-password",
    DB:       0,
    PoolSize: 20,
}

client, err := redis.NewClient(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Redis Cluster

```go
cfg := redis.Config{
    Type: redis.ClientTypeCluster,
    ClusterAddrs: []string{
        "redis-node1:6379",
        "redis-node2:6379",
        "redis-node3:6379",
    },
    Password: "cluster-password",
}

client, err := redis.NewClient(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Redis Sentinel

```go
cfg := redis.Config{
    Type:       redis.ClientTypeSentinel,
    MasterName: "mymaster",
    SentinelAddrs: []string{
        "sentinel1:26379",
        "sentinel2:26379",
        "sentinel3:26379",
    },
    Password:         "redis-password",
    SentinelPassword: "sentinel-password",
    DB:               0,
}

client, err := redis.NewClient(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### With TLS

```go
cfg := redis.Config{
    URL:           "redis://secure-redis.example.com:6380",
    Type:          redis.ClientTypeSingleNode,
    Password:      "my-password",
    EnableTLS:     true,
    TLSServerName: "secure-redis.example.com",
}

client, err := redis.NewClient(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### With Namespace

```go
cfg := redis.Config{
    URL:       "redis://localhost:6379",
    Type:      redis.ClientTypeSingleNode,
    Namespace: "myapp",
}

client, err := redis.NewClient(ctx, cfg)
if err != nil {
    log.Fatal(err)
}

// Keys will be automatically prefixed with "myapp:"
key := client.WithNamespace("user:123") // Returns "myapp:user:123"
```

## Functional Options

The package supports functional options for additional configuration:

```go
client, err := redis.NewClient(
    ctx,
    cfg,
    redis.WithConnectTimeout(10*time.Second),
    redis.WithMaxRetries(5),
    redis.WithRetryCallback(func(attempt int, err error) {
        log.Printf("Retry attempt %d: %v", attempt, err)
    }),
    redis.WithOnConnect(func(ctx context.Context, client *redis.Client) error {
        log.Println("Connected to Redis")
        return nil
    }),
)
```

Available options:

- `WithConnectTimeout(timeout time.Duration)` - Set connection timeout
- `WithMaxRetries(maxRetries int)` - Set maximum retry attempts
- `WithRetryDelay(delay time.Duration)` - Set retry delay
- `WithRetryCallback(func(attempt int, err error))` - Set retry callback
- `WithOnConnect(func(ctx context.Context, client *Client) error)` - Set post-connect hook
- `WithOnDisconnect(func(client *Client) error)` - Set pre-disconnect hook

## Metrics and Statistics

### Pool Statistics

```go
stats := client.Stats()
if stats != nil {
    fmt.Printf("Total Connections: %d\n", stats.TotalConns)
    fmt.Printf("Idle Connections: %d\n", stats.IdleConns)
    fmt.Printf("Hits: %d\n", stats.Hits)
    fmt.Printf("Misses: %d\n", stats.Misses)
}
```

### Application Metrics

Enable metrics collection:

```go
cfg := redis.Config{
    URL:           "redis://localhost:6379",
    Type:          redis.ClientTypeSingleNode,
    EnableMetrics: true,
}

client, err := redis.NewClient(ctx, cfg)

// Get metrics
metrics := client.GetMetrics()
if metrics != nil {
    fmt.Printf("Commands Executed: %d\n", metrics.CommandsExecuted)
    fmt.Printf("Commands Failed: %d\n", metrics.CommandsFailed)
    fmt.Printf("Average Latency: %v\n", metrics.AverageLatency)
}
```

## Error Handling

The package provides custom error types for better error handling:

```go
client, err := redis.NewClient(ctx, cfg)
if err != nil {
    switch {
    case errors.Is(err, redis.ErrInvalidConfig):
        log.Println("Invalid configuration")
    case errors.Is(err, redis.ErrConnectionFailed):
        log.Println("Connection failed")
    case errors.Is(err, redis.ErrPingFailed):
        log.Println("Ping failed")
    default:
        log.Printf("Unexpected error: %v", err)
    }
}
```

Available errors:

- `ErrInvalidConfig` - Invalid configuration
- `ErrConnectionFailed` - Connection attempt failed
- `ErrMissingURL` - Redis URL is required
- `ErrInvalidClientType` - Invalid client type
- `ErrEmptyClientType` - Client type is empty
- `ErrPingFailed` - Ping operation failed
- `ErrInvalidDB` - Invalid DB number
- `ErrClientClosed` - Client is already closed

## Uber FX Integration

### Using Module

```go
app := fx.New(
    fx.Provide(
        func() redis.Config {
            return redis.Config{
                URL:  "redis://localhost:6379",
                Type: redis.ClientTypeSingleNode,
            }
        },
    ),
    redis.Module, // Provides *redis.Client
    fx.Invoke(func(client *redis.Client) {
        // Use client
    }),
)
```

### Using UniversalClientModule

```go
app := fx.New(
    fx.Provide(
        func() redis.Config {
            return redis.Config{
                URL:  "redis://localhost:6379",
                Type: redis.ClientTypeSingleNode,
            }
        },
    ),
    redis.UniversalClientModule, // Provides redis.UniversalClient
    fx.Invoke(func(client redis.UniversalClient) {
        // Use UniversalClient interface
    }),
)
```

### With Options

```go
app := fx.New(
    fx.Provide(
        func() redis.Config {
            return redis.Config{
                URL:  "redis://localhost:6379",
                Type: redis.ClientTypeSingleNode,
            }
        },
        fx.Annotate(
            func() []redis.Option {
                return []redis.Option{
                    redis.WithMaxRetries(5),
                    redis.WithRetryCallback(func(attempt int, err error) {
                        log.Printf("Retry %d: %v", attempt, err)
                    }),
                }
            },
            fx.ResultTags(`group:"redis_options"`),
        ),
    ),
    redis.Module,
)
```

## Best Practices

1. **Always use context**: Pass context to all Redis operations for proper cancellation and timeout handling
2. **Configure timeouts**: Set appropriate timeouts based on your application's requirements
3. **Use connection pooling**: Configure pool size based on your expected load
4. **Enable metrics**: Monitor your Redis usage with built-in metrics
5. **Handle errors properly**: Use type assertions to handle specific error types
6. **Use namespaces**: Isolate keys in multi-tenant applications
7. **Close connections**: Always defer `client.Close()` to ensure proper cleanup
8. **TLS in production**: Enable TLS for production deployments

## Thread Safety

The Redis client is thread-safe and can be used concurrently from multiple goroutines.

## License

This package is part of the bricks library and follows the same license.
