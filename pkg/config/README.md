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

1. **Config directory**: Reads `APP_CONFIG_DIR` (defaults to `./config` if not set)
2. **Loading order**: `base.yaml` is loaded first, then the environment-specific file (e.g., `local.yaml`) merges on top
3. **Environment detection**: Reads `APP_ENV` environment variable (defaults to `local` if not set)
4. **Environment variable overrides**: Any `APP_` prefixed env var overrides config values after files are loaded
5. **Unmarshal**: Final merged config is unmarshaled into your struct using the `config` struct tag

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

// Load configuration from default config directory (./config)
cfg, err := config.New[AppConfig]()
if err != nil {
    log.Fatal(err)
}

// Access the typed config value
appName := cfg.Get().App.Name
appPort := cfg.Get().App.Port
```

**Important**: The returned value is `Config[AppConfig]`, not `AppConfig` directly. Use `.Get()` to access the actual config struct.

## File Structure

Configuration files must be placed in a directory with this structure.
By default, the directory is `./config` (relative to the process root) and can be overridden with `APP_CONFIG_DIR`.

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

## Config Directory Selection

The config directory is resolved via `APP_CONFIG_DIR` and defaults to `./config` when not set.
The resolved path is checked under the process root and must exist:

```bash
export APP_CONFIG_DIR=./config
```

## Environment Variable Overrides

Any configuration value can be overridden using environment variables with the `APP_` prefix.

**Transformation rule**: `APP_<PATH>` where path uses double underscore (`__`) as the nesting delimiter. All keys are automatically mapped under the `app.` root.

**Why double underscore?** Single underscores inside key names (e.g., `api_key`, `max_tokens`) are preserved. Only double underscores are converted to dots for nesting.

Examples:
```bash
# Override app.name
export APP_NAME=MyServiceProd

# Override app.database.host
export APP_DATABASE__HOST=prod-db.example.com

# Override app.redis.password
export APP_REDIS__PASSWORD=secret

# Override app.auth.jwt_secret
export APP_AUTH__JWT_SECRET=my-secret

# Override deeply nested values (app.ai.providers.openai.api_key)
export APP_AI__PROVIDERS__OPENAI__API_KEY=sk-xxxx

# Keys with single underscores in names are preserved (app.database.api_key)
export APP_DATABASE__API_KEY=secret123
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
    config.WithPath("app.database"),  // path to subtree
)
// cfg.Get() now contains only the database config

// Load only the redis section
type RedisConfig struct {
    Host string `config:"host"`
    Port int    `config:"port"`
}

redisCfg, err := config.New[RedisConfig](
    config.WithPath("app.redis"),  // path to subtree
)
```

**Path format**: Use dot notation to navigate nested structures (e.g., `"app.database"`, `"features.auth"`).

## Fx Integration

Use `Provide` to inject config into fx modules:

```go
import (
    "go.uber.org/fx"
    "github.com/cristiano-pacheco/bricks/pkg/config"
)

type DatabaseConfig struct {
    Host string `config:"host"`
    Port int    `config:"port"`
}

var Module = fx.Module("database",
    config.Provide[DatabaseConfig]("app.database"),
    fx.Invoke(func(cfg config.Config[DatabaseConfig]) {
        db := cfg.Get()
        // use db.Host, db.Port
    }),
)
```

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
    
    cfg, err := config.New[AppConfig]()
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

With `APP_ENV=production` and `APP_PORT=8443`:
- Outputs: "Starting MyService on port 8443"
- Database host will be "prod-db.example.com" (from production.yaml)
- Port 8443 from env var (`APP_PORT` → `app.port`) overrides production.yaml's 443
