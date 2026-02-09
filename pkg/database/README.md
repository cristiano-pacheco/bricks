# Database Module

PostgreSQL database connection module with GORM and Uber FX integration.

## Configuration

For a complete configuration example, see [config/config.yaml](config/config.yaml).

## Installation

Install the entire Bricks framework:

```bash
go get github.com/cristiano-pacheco/bricks@latest
```

Then import the database module:

```go
import "github.com/cristiano-pacheco/bricks/pkg/database"
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
    "log"

    "github.com/cristiano-pacheco/bricks/pkg/database"
)

func main() {
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

    db, err := database.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer func() {
        if sqlDB, err := db.DB(); err == nil {
            sqlDB.Close()
        }
    }()

    // Use database directly with GORM
    var users []User
    db.Find(&users)
}
```

### With Uber Fx

```go
package main

import (
    "github.com/cristiano-pacheco/bricks/pkg/database"
    "go.uber.org/fx"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func main() {
    fx.New(
        fx.Provide(
            func() database.Config {
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
            },
            database.NewWithLifecycle,
        ),
        fx.Invoke(func(db *gorm.DB) {
            // Use database - connection is automatically closed on shutdown
            var users []User
            db.Find(&users)
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

## API Reference

### Core Functions

#### New

```go
func New(cfg Config) (*gorm.DB, error)
```

Creates a new database connection with automatic retry and connection pool configuration.
Returns a `*gorm.DB` instance ready to use. The caller is responsible for closing the connection.

**Example:**
```go
db, err := database.New(cfg)
if err != nil {
    return err
}
defer func() {
    if sqlDB, err := db.DB(); err == nil {
        sqlDB.Close()
    }
}()
```

#### NewWithLifecycle

```go
func NewWithLifecycle(cfg Config, lc fx.Lifecycle) (*gorm.DB, error)
```

Creates a new database connection with fx.Lifecycle management.
The connection is automatically closed when the application stops via OnStop hook.

**Example with Fx:**
```go
fx.New(
    fx.Provide(
        func() database.Config { return cfg },
        database.NewWithLifecycle,
    ),
    fx.Invoke(func(db *gorm.DB) {
        // use db
    }),
)
```

### Working with GORM

Once you have a `*gorm.DB` instance, you can use all GORM features directly:

```go
// Health check
sqlDB, err := db.DB()
if err != nil {
    return err
}
if err := sqlDB.PingContext(ctx); err != nil {
    return err
}

// Connection statistics
stats := sqlDB.Stats()
log.Printf("Open: %d, InUse: %d, Idle: %d", 
    stats.OpenConnections, stats.InUse, stats.Idle)

// Close connection
sqlDB.Close()
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
db, err := database.New(cfg)
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
- Context cancellation support (10 second timeout)

The retry mechanism is built-in and requires no configuration.

## Fx Integration

Use `database.NewWithLifecycle` for seamless Fx integration:

```go
fx.New(
    fx.Provide(
        func() database.Config {
            return database.Config{
                Host: "localhost",
                Port: 5432,
                Name: "mydb",
                User: "user",
            }
        },
        database.NewWithLifecycle,
    ),
    fx.Invoke(func(db *gorm.DB) {
        // Use db - automatically closed on shutdown
    }),
)
```

Features:
- Automatic connection cleanup on shutdown via OnStop hook
- Connection pool configuration
- No need for manual Close() calls

## Connection Pool Management

The module automatically configures connection pooling with sensible defaults:

- **Max Connection Lifetime**: 1 hour (configurable via `WithConnMaxLifetime`)
- **Max Idle Time**: 10 minutes (configurable via `WithConnMaxIdleTime`)
- **Max Open Connections**: Set via `Config.MaxOpenConnections`
- **Max Idle Connections**: Set via `Config.MaxIdleConnections`

Monitor pool health using the underlying `sql.DB`:

```go
sqlDB, err := db.DB()
if err != nil {
    log.Fatal(err)
}

stats := sqlDB.Stats()
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

1. **Use NewWithLifecycle with Fx**: When using Fx, prefer `NewWithLifecycle` for automatic lifecycle management
2. **Close connections**: When using `New()`, always close the connection properly:
   ```go
   defer func() {
       if sqlDB, err := db.DB(); err == nil {
           sqlDB.Close()
       }
   }()
   ```
3. **Health checks**: Use the underlying sql.DB for health checks:
   ```go
   sqlDB, _ := db.DB()
   sqlDB.PingContext(ctx)
   ```
4. **Monitor statistics**: Check connection pool stats via `sqlDB.Stats()` to detect issues
5. **Configure timeouts**: Set appropriate statement and lock timeouts in the Config for your use case
6. **Use connection limits**: Configure `MaxOpenConnections` and `MaxIdleConnections` based on your load
7. **Enable logging in development**: Set `EnableLogs: true` and provide a Logger for debugging

## Testing

Run tests with:

```bash
go test ./...
```

Note: Integration tests require a running PostgreSQL instance.

## License

MIT