# Database Module

PostgreSQL database connection module with GORM and Uber FX integration.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks/database@latest
```

## Features

- PostgreSQL connection using GORM
- Connection pool management with configurable timeouts
- Automatic retry with exponential backoff
- Structured logging support
- Uber FX lifecycle integration
- Automatic connection cleanup
- Health check support
- Connection statistics monitoring
- SSL/TLS support

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/cristiano-pacheco/bricks/database"
)

func main() {
    ctx := context.Background()
    
    cfg := database.Config{
        Host:               "localhost",
        User:               "myuser",
        Password:           "mypassword",
        Name:               "mydb",
        Port:               5432,
        MaxOpenConnections: 25,
        MaxIdleConnections: 5,
        SSLMode:            false,
        PrepareSTMT:        true,
        EnableLogs:         true,
    }

    client, err := database.NewClient(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get GORM DB instance
    db := client.DB()

    // Use database
    var users []User
    db.Find(&users)
}
```

### With Functional Options

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/cristiano-pacheco/bricks/database"
)

func main() {
    ctx := context.Background()
    
    cfg := database.Config{
        Host:               "localhost",
        User:               "myuser",
        Password:           "mypassword",
        Name:               "mydb",
        Port:               5432,
        MaxOpenConnections: 25,
        MaxIdleConnections: 5,
    }

    client, err := database.NewClient(ctx, cfg,
        database.WithConnectTimeout(10*time.Second),
        database.WithMaxRetries(5),
        database.WithRetryDelay(2*time.Second),
        database.WithConnMaxLifetime(1*time.Hour),
        database.WithConnMaxIdleTime(10*time.Minute),
        database.WithRetryCallback(func(attempt int, err error) {
            log.Printf("Retry attempt %d: %v", attempt, err)
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Check connection health
    if err := client.Ping(ctx); err != nil {
        log.Fatal("Database health check failed:", err)
    }

    // Get connection statistics
    stats, err := client.Stats()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Open connections: %d/%d", stats.OpenConnections, stats.MaxOpenConnections)

    db := client.DB()
    // Use database...
}
```

### With Uber Fx (GORM DB)

```go
package main

import (
    "github.com/cristiano-pacheco/bricks/database"
    "go.uber.org/fx"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func main() {
    fx.New(
        database.Module,
        fx.Provide(func() database.Config {
            return database.Config{
                Host:               "localhost",
                User:               "myuser",
                Password:           "mypassword",
                Name:               "mydb",
                Port:               5432,
                MaxOpenConnections: 25,
                MaxIdleConnections: 5,
                SSLMode:            false,
                PrepareSTMT:        true,
                EnableLogs:         true,
                Logger:             logger.Default.LogMode(logger.Info),
            }
        }),
        fx.Invoke(func(db *gorm.DB) {
            // Use database
            var users []User
            db.Find(&users)
        }),
    ).Run()
}
```

### With Uber Fx (Full Client)

```go
package main

import (
    "context"

    "github.com/cristiano-pacheco/bricks/database"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        database.ClientModule,
        fx.Provide(func() database.Config {
            return database.Config{
                Host:     "localhost",
                User:     "myuser",
                Password: "mypassword",
                Name:     "mydb",
                Port:     5432,
            }
        }),
        fx.Invoke(func(client *database.Client) {
            ctx := context.Background()
            
            // Health check
            if err := client.Ping(ctx); err != nil {
                panic(err)
            }
            
            // Get statistics
            stats, _ := client.Stats()
            log.Printf("Connections: %d in use, %d idle", stats.InUse, stats.Idle)
            
            // Use GORM DB
            db := client.DB()
            db.AutoMigrate(&User{})
        }),
    ).Run()
}
```

## Configuration

### Config Struct

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| **Connection** |
| Host | string | Database host | Required |
| Port | uint | Database port | Required |
| Name | string | Database name | Required |
| User | string | Database user | Required |
| Password | string | Database password | - |
| **SSL Configuration** |
| SSLMode | bool | Enable SSL mode | false |
| SSLCert | string | Path to SSL certificate | - |
| SSLKey | string | Path to SSL key | - |
| SSLRootCert | string | Path to SSL root certificate | - |
| **Connection Pool** |
| MaxOpenConnections | int | Maximum number of open connections | - |
| MaxIdleConnections | int | Maximum number of idle connections | - |
| **GORM Settings** |
| PrepareSTMT | bool | Enable prepared statement cache | false |
| SkipDefaultTransaction | bool | Skip default transaction for single create, update, delete | false |
| DisableForeignKeyConstraint | bool | Disable foreign key constraints when migrating | false |
| **Logging** |
| EnableLogs | bool | Enable SQL logging | false |
| Logger | logger.Interface | Custom GORM logger | - |
| NamingStrategy | schema.Namer | Custom naming strategy | - |
| **Advanced Settings** |
| TimeZone | string | Database timezone | UTC |
| ApplicationName | string | Application name in connection | - |
| SearchPath | string | PostgreSQL search path | - |
| StatementTimeout | int | Statement timeout (milliseconds) | - |
| LockTimeout | int | Lock timeout (milliseconds) | - |
| IdleInTransaction | int | Idle in transaction timeout (ms) | - |
| ConnectTimeout | int | Connection timeout (seconds) | - |
| PreferSimpleProtol | bool | Prefer simple protocol | false |

### Functional Options

| Option | Description | Default |
|--------|-------------|---------|
| WithConnectTimeout(duration) | Sets the connection timeout | 10s |
| WithMaxRetries(int) | Sets maximum connection retry attempts | 3 |
| WithRetryDelay(duration) | Sets base delay between retries | 1s |
| WithConnMaxLifetime(duration) | Sets maximum connection lifetime | 1h |
| WithConnMaxIdleTime(duration) | Sets maximum connection idle time | 10m |
| WithRetryCallback(func) | Sets callback for retry attempts | - |

## API Reference

### Client

```go
type Client struct {
    // contains filtered or unexported fields
}
```

#### NewClient

```go
func NewClient(ctx context.Context, cfg Config, opts ...Option) (*Client, error)
```

Creates a new database client with context support and functional options.

#### Methods

```go
func (c *Client) DB() *gorm.DB
```

Returns the underlying GORM database instance.

```go
func (c *Client) Close() error
```

Closes the database connection.

```go
func (c *Client) Ping(ctx context.Context) error
```

Checks if the database connection is alive.

```go
func (c *Client) Stats() (ConnectionStats, error)
```

Returns database connection statistics.

### ConnectionStats

```go
type ConnectionStats struct {
    MaxOpenConnections int           // Maximum number of open connections
    OpenConnections    int           // Number of established connections
    InUse              int           // Number of connections in use
    Idle               int           // Number of idle connections
    WaitCount          int64         // Total connections waited for
    WaitDuration       time.Duration // Total time blocked waiting
    MaxIdleClosed      int64         // Connections closed due to max idle
    MaxLifetimeClosed  int64         // Connections closed due to max lifetime
}
```

### DSN Generation

```go
func (c Config) DSN() string
```

Generates a GORM-compatible DSN string.

```go
func (c Config) PostgresDSN() (string, error)
```

Generates a standard PostgreSQL connection string (postgres://).

### Helper Methods

```go
func (c Config) Validate() error
```

Validates the database configuration.

```go
func (c Config) Clone() Config
```

Creates a deep copy of the configuration.

```go
func (c Config) WithDatabase(name string) Config
```

Creates a new config with a different database name.

## Error Handling

The module provides predefined errors:

- `ErrInvalidConfig` - Invalid or incomplete configuration
- `ErrConnectionFailed` - Database connection attempt failed
- `ErrInvalidPortNumber` - Port number exceeds valid range
- `ErrMissingHost` - Database host is required
- `ErrMissingName` - Database name is required
- `ErrMissingUser` - Database user is required
- `ErrMissingPort` - Database port is required

- `ErrMissingPort` - Database port is required

Example error handling:

```go
client, err := database.NewClient(ctx, cfg)
if err != nil {
    if errors.Is(err, database.ErrConnectionFailed) {
        log.Fatal("Cannot connect to database:", err)
    }
    if errors.Is(err, database.ErrInvalidConfig) {
        log.Fatal("Invalid configuration:", err)
    }
    log.Fatal("Unexpected error:", err)
}
```

## Retry Mechanism

The module implements automatic retry with exponential backoff:

- Default retry attempts: 3
- Default base delay: 1 second
- Exponential backoff formula: baseDelay * 2^(attempt-1)
- Maximum backoff: 30 seconds
- Context cancellation support

Customize retry behavior:

```go
client, err := database.NewClient(ctx, cfg,
    database.WithMaxRetries(5),
    database.WithRetryDelay(2*time.Second),
    database.WithRetryCallback(func(attempt int, err error) {
        log.Printf("Connection attempt %d failed: %v", attempt, err)
    }),
)
```

## Fx Integration

The module provides two Fx modules:

### database.Module

Provides `*gorm.DB` with automatic lifecycle management:
- Connection pool configuration on start
- Health check via Ping on start
- Automatic connection cleanup on shutdown

### database.ClientModule

Provides `*database.Client` with full API access:
- All features of Module
- Access to Client methods (Ping, Stats, Close)
- Enhanced control over the connection

Both modules require a `database.Config` provider and optionally accept `[]database.Option` for advanced configuration.

## Connection Pool Management

The module automatically configures connection pooling with sensible defaults:

- **Max Connection Lifetime**: 1 hour (configurable via `WithConnMaxLifetime`)
- **Max Idle Time**: 10 minutes (configurable via `WithConnMaxIdleTime`)
- **Max Open Connections**: Set via `Config.MaxOpenConnections`
- **Max Idle Connections**: Set via `Config.MaxIdleConnections`

Monitor pool health using `Client.Stats()`:

```go
stats, err := client.Stats()
if err != nil {
    log.Fatal(err)
}

log.Printf("Connection Pool Stats:")
log.Printf("  Open: %d/%d", stats.OpenConnections, stats.MaxOpenConnections)
log.Printf("  In Use: %d", stats.InUse)
log.Printf("  Idle: %d", stats.Idle)
log.Printf("  Wait Count: %d", stats.WaitCount)
log.Printf("  Wait Duration: %v", stats.WaitDuration)
```

## Advanced Configuration

### SSL/TLS Configuration

```go
cfg := database.Config{
    Host:        "localhost",
    Port:        5432,
    Name:        "mydb",
    User:        "myuser",
    Password:    "mypassword",
    SSLMode:     true,
    SSLCert:     "/path/to/client-cert.pem",
    SSLKey:      "/path/to/client-key.pem",
    SSLRootCert: "/path/to/ca-cert.pem",
}
```

### Custom Logger

```go
import "gorm.io/gorm/logger"

cfg := database.Config{
    Host:       "localhost",
    Port:       5432,
    Name:       "mydb",
    User:       "myuser",
    Password:   "mypassword",
    EnableLogs: true,
    Logger:     logger.Default.LogMode(logger.Info),
}
```

### Custom Naming Strategy

```go
import "gorm.io/gorm/schema"

cfg := database.Config{
    Host:     "localhost",
    Port:     5432,
    Name:     "mydb",
    User:     "myuser",
    Password: "mypassword",
    NamingStrategy: schema.NamingStrategy{
        TablePrefix:   "t_",
        SingularTable: true,
    },
}
```

### PostgreSQL Specific Settings

```go
cfg := database.Config{
    Host:              "localhost",
    Port:              5432,
    Name:              "mydb",
    User:              "myuser",
    Password:          "mypassword",
    ApplicationName:   "my-app",
    SearchPath:        "public,custom_schema",
    StatementTimeout:  30000, // 30 seconds
    LockTimeout:       5000,  // 5 seconds
    IdleInTransaction: 60000, // 1 minute
    ConnectTimeout:    10,    // 10 seconds
    TimeZone:          "America/Sao_Paulo",
}
```

## Best Practices

1. **Always use context**: Pass appropriate context to `NewClient` and `Ping` for timeout control
2. **Close connections**: Always defer `client.Close()` to ensure proper cleanup
3. **Health checks**: Use `Ping()` to verify connectivity before critical operations
4. **Monitor statistics**: Regularly check `Stats()` to detect connection pool issues
5. **Configure timeouts**: Set appropriate statement and lock timeouts for your use case
6. **Use connection limits**: Configure `MaxOpenConnections` and `MaxIdleConnections` based on your load
7. **Enable retry callback**: Use `WithRetryCallback` for better observability in production

## Testing

Run tests with:

```bash
go test ./...
```

Note: Integration tests require a running PostgreSQL instance.

## License

MIT