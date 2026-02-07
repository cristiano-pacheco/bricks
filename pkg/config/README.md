# Config

Type-safe configuration loader with multi-environment support and environment variable overrides.

## Overview

This package loads YAML configuration files and unmarshals them into strongly-typed Go structs using generics. It supports:

- **Multi-environment configs**: Merge base.yaml with environment-specific files (local.yaml, production.yaml, etc.)
- **Environment variables**: Override any config value using `APP_` prefixed env vars
- **Type safety**: Generic `Config[T]` type ensures compile-time type checking
- **Partial loading**: Load only a subtree of the config using `WithPath` option

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## How It Works

1. **Loading order**: `base.yaml` is loaded first, then the environment-specific file (e.g., `local.yaml`) merges on top
2. **Environment detection**: Reads `APP_ENV` environment variable (defaults to `local` if not set)
3. **Environment variable overrides**: Any `APP_` prefixed env var overrides config values after files are loaded
4. **Unmarshal**: Final merged config is unmarshaled into your struct using the `config` struct tag

## Basic Usage

Define a struct with `config` tags matching your YAML structure:

```go
import (
    "log"
    "github.com/cristiano-pacheco/bricks/pkg/config"
)

type AppConfig struct {
    App struct {
        Name string `config:"name"`  // maps to YAML key "name"
        Port int    `config:"port"`  // maps to YAML key "port"
    } `config:"app"`  // maps to YAML top-level key "app"
}

// Load configuration from ./config directory
cfg, err := config.New[AppConfig]("./config")
if err != nil {
    log.Fatal(err)
}

// Access the typed config value
appName := cfg.Get().App.Name
appPort := cfg.Get().App.Port
```

**Important**: The returned value is `Config[AppConfig]`, not `AppConfig` directly. Use `.Get()` to access the actual config struct.

## File Structure

Configuration files must be placed in a directory with this structure:

```
config/
├── base.yaml        # Required: base configuration applied to all environments
├── local.yaml       # Optional: overrides for local development
├── production.yaml  # Optional: overrides for production
└── staging.yaml     # Optional: overrides for staging
```

**Example base.yaml**:
```yaml
app:
  name: "MyApp"
  port: 8080
  debug: false
database:
  host: "localhost"
  port: 5432
```

**Example production.yaml** (only overrides):
```yaml
app:
  port: 443
  debug: false
database:
  host: "prod-db.example.com"
```

The final config in production will merge both files, with production.yaml values taking precedence.

## Environment Selection

The active environment is determined by the `APP_ENV` environment variable:

```bash
export APP_ENV=production  # loads base.yaml + production.yaml
export APP_ENV=local       # loads base.yaml + local.yaml (default)
export APP_ENV=staging     # loads base.yaml + staging.yaml
```

If `APP_ENV` is not set, it defaults to `local`.

## Environment Variable Overrides

Any configuration value can be overridden using environment variables with the `APP_` prefix.

**Transformation rule**: `APP_<PATH>` where path uses underscores and is case-insensitive.

Examples:
```bash
# Override app.port (nested: app -> port)
export APP_APP_PORT=9000

# Override database.host (nested: database -> host)
export APP_DATABASE_HOST=custom-db.example.com

# Override deeply nested values
export APP_APP_FEATURE_ENABLED=true  # overrides app.feature.enabled
```

**Precedence order** (highest to lowest):
1. Environment variables (`APP_*`)
2. Environment-specific YAML file (e.g., `production.yaml`)
3. Base YAML file (`base.yaml`)

## Struct Tags

Use the `config` struct tag to map struct fields to YAML keys:

```go
type Config struct {
    // Maps to YAML: server.hostname
    Server struct {
        Hostname string `config:"hostname"`
        Port     int    `config:"port"`
    } `config:"server"`
    
    // Maps to YAML: features.auth.enabled
    Features struct {
        Auth struct {
            Enabled bool `config:"enabled"`
        } `config:"auth"`
    } `config:"features"`
}
```

**Note**: The tag name must match the YAML key exactly (case-sensitive).

## Load Config Subtree

Use `WithPath` to load only a portion of the config file into a struct:

```go
// Full config structure in YAML
/*
app:
  name: "MyApp"
  database:
    host: "localhost"
    port: 5432
redis:
  host: "localhost"
  port: 6379
*/

// Load only the database section
type DatabaseConfig struct {
    Host string `config:"host"`
    Port int    `config:"port"`
}

cfg, err := config.New[DatabaseConfig](
    "./config",
    config.WithPath("app.database"),  // path to subtree
)
// cfg.Get() now contains only the database config

// Load only the redis section
type RedisConfig struct {
    Host string `config:"host"`
    Port int    `config:"port"`
}

redisCfg, err := config.New[RedisConfig](
    "./config",
    config.WithPath("redis"),  // path to subtree
)
```

**Path format**: Use dot notation to navigate nested structures (e.g., `"app.database"`, `"features.auth"`).

## Complete Example

**config/base.yaml**:
```yaml
app:
  name: "MyService"
  port: 8080
  debug: true
  database:
    host: "localhost"
    port: 5432
    name: "mydb"
```

**config/production.yaml**:
```yaml
app:
  port: 443
  debug: false
  database:
    host: "prod-db.example.com"
```

**main.go**:
```go
package main

import (
    "fmt"
    "log"
    "github.com/cristiano-pacheco/bricks/pkg/config"
)

type AppConfig struct {
    App struct {
        Name  string `config:"name"`
        Port  int    `config:"port"`
        Debug bool   `config:"debug"`
        Database struct {
            Host string `config:"host"`
            Port int    `config:"port"`
            Name string `config:"name"`
        } `config:"database"`
    } `config:"app"`
}

func main() {
    // Set environment (or use APP_ENV env var)
    // os.Setenv("APP_ENV", "production")
    
    cfg, err := config.New[AppConfig]("./config")
    if err != nil {
        log.Fatal(err)
    }
    
    app := cfg.Get().App
    fmt.Printf("Starting %s on port %d\n", app.Name, app.Port)
    fmt.Printf("Database: %s:%d/%s\n", 
        app.Database.Host, 
        app.Database.Port, 
        app.Database.Name,
    )
}
```

With `APP_ENV=production` and `APP_APP_PORT=8443`:
- Outputs: "Starting MyService on port 8443"
- Database host will be "prod-db.example.com" (from production.yaml)
- Port 8443 from env var overrides production.yaml's 443
