# Config Package

A robust and lightweight configuration management package for Go applications, built on top of [Koanf](https://github.com/knadh/koanf), with support for **generics**, multiple environments, and automatic configuration loading.

## Features

- 🚀 **Generics Support**: Type-safe config loading with `Load[T]()` - simple and elegant
- ⚡ **Lightweight & Fast**: Built on Koanf - modern, performant, and modular
- 🔄 **Auto Environment Detection**: Automatically loads config based on `APP_ENV`
- 📁 **Multi-Environment Support**: Load base config and override with environment-specific configs
- 🔄 **Automatic Merging**: Base config + environment config = final configuration
- 📦 **YAML Support**: Native YAML configuration files
- 🌍 **Environment Variables**: Override any config with `APP_*` env vars
- 🎯 **Type-Safe**: Strong typing with struct unmarshaling and generics
- 🔌 **Uber FX Integration**: First-class support for dependency injection
- ⚙️ **Flexible**: Access values by key or unmarshal to structs
- 🛡️ **Validated**: Built-in error handling and validation
- 🧪 **Easy Testing**: Simple API designed for testability

## Why Koanf over Viper?

- **More lightweight**: Modular architecture, import only what you need
- **Better performance**: Less overhead, faster config loading
- **Simpler API**: Cleaner and more intuitive interface
- **Modern design**: Built with Go best practices
- **Perfect for generics**: Better type-safety support

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Quick Start

### Directory Structure

```
project/
├── config/
│   ├── base.yaml      # Base configuration (required)
│   ├── local.yaml     # Local development overrides
│   ├── staging.yaml   # Staging environment overrides
│   ├── production.yaml # Production environment overrides
│   └── homol.yaml     # Homologation environment overrides
└── main.go
```

### Configuration Files

**config/base.yaml** (Base configuration):
```yaml
app:
  name: "MyApp"
  port: 8080
  debug: false
  
database:
  host: "localhost"
  port: 5432
  name: "mydb"
  user: "postgres"
  max_connections: 25
  
redis:
  host: "localhost"
  port: 6379
  db: 0
```

**config/local.yaml** (Local overrides):
```yaml
app:
  debug: true
  
database:
  password: "local_password"
  
redis:
  password: "local_redis_pass"
```

**config/production.yaml** (Production overrides):
```yaml
app:
  port: 443
  
database:
  host: "prod-db.example.com"
  max_connections: 100
  ssl: true
  
redis:
  host: "prod-redis.example.com"
```

## Usage with Generics (Recommended)

The simplest and most type-safe way to load configuration. Environment is automatically detected from `APP_ENV` or defaults to `"local"`:

```go
package main

import (
    "log"
    
    "github.com/cristiano-pacheco/bricks/pkg/config"
)

// Define your config struct with koanf tags
type AppConfig struct {
    App struct {
        Name  string `koanf:"name"`
        Port  int    `koanf:"port"`
        Debug bool   `koanf:"debug"`
    } `koanf:"app"`
    
    Database struct {
        Host     string `koanf:"host"`
        Port     int    `koanf:"port"`
        User     string `koanf:"user"`
        Password string `koanf:"password"`
    } `koanf:"database"`
}

func main() {
    // Load config with generics - automatically detects environment!
    // Reads from APP_ENV or defaults to "local"
    cfg, err := config.Load[AppConfig]("./config")
    if err != nil {
        log.Fatal(err)
    }
    
    // Use your config with full type safety
    log.Printf("Starting %s on port %d", cfg.App.Name, cfg.App.Port)
    log.Printf("Database: %s@%s:%d", cfg.Database.User, cfg.Database.Host, cfg.Database.Port)
}
```

### Explicit Environment

If you need to specify the environment explicitly:

```go
// Load with specific environment
cfg, err := config.LoadEnv[AppConfig]("./config", "production")
if err != nil {
    log.Fatal(err)
}
```

### Environment Variables

Any

### MustLoad for Critical Config

```go
// Panics if config cannot be loaded - perfect for startup
func main() {
    cfg := config.MustLoad[AppConfig]("./config")
    
    // If we reach here, config is valid
    startServer(cfg)
}
```

## Manual Configuration Access

If you prefer not to use generics, you can access config values directly:
```go
// Panics if config cannot be loaded - perfect for startup
func main() {
    cfg := config.MustLoad[AppConfig]("./config", "production")
    
    // If we reach here, config is valid
    startServer(cfg)
}
```

### Basic Usage
## Manual Configuration Access

If you prefer not to use generics, you can access config values directly:

```go
// Create config instance
cfg, err := config.New("./config", "local")
if err != nil {
    log.Fatal(err)
}

// Access individual values
appName := cfg.GetString("app.name")
port := cfg.GetInt("app.port")
debug := cfg.GetBool("app.debug")

// Or unmarshal to struct manually
type AppConfig struct {
    App struct {
        Name  string `koanf:"name"`
        Port  int    `koanf:"port"`
        Debug bool   `koanf:"debug"`
    } `koanf:"app"`
}

var appConfig AppConfig
if err := cfg.Unmarshal(&appConfig); err != nil {
    log.Fatal(err)
}
```

## Uber FX Integration

### Using Generics (Recommended)

```go
package mainkoanf:"name"`
        Port int    `koanf:"port"`
    } `koanf:"app"`
}

func main() {
    fx.New(
        // Provide config dir (optional - defaults to "./config")
        config.ProvideConfigDir("./config"),
        
        // Provide environment (optional - auto-detected from APP_ENV)
        // config.ProvideEnvironment("production
    App struct {
        Name string `mapstructure:"name"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"app"`
}

func main() {
    fx.New(
        // Provide config dir and environment
        config.ProvideConfigDir("./config"),
        config.ProvideEnvironment("local"),
        
        // Use the config module
        config.Module,
        
        // Provide typed config using generics
        config.ProvideConfig[AppConfig](),
        
        // Inject typed config directly
        fx.Invoke(func(cfg AppConfig) {
            log.Printf("Starting %s on port %d", cfg.App.Name, cfg.App.Port)
        }),
    ).Run()
}
```
### Using Generics (Recommended)

```go
package main

import (
    "log"
    
    "github.com/cristiano-pacheco/bricks/pkg/config"
    "go.uber.org/fx"
)

type AppConfig struct {
### With Custom Config Directory or Environment

```go
fx.New(
    // Optional: Override config directory (defaults to "./config")
    config.ProvideConfigDir("/etc/myapp/config"),
    
    // Optional: Override environment (defaults to APP_ENV or "local")
    config.ProvideEnvironment("production"),
    
    config.Module,
    config.ProvideConfig[AppConfig](),
    
    fx.Invoke(func(cfg AppConfig) {
        log.Printf("Starting %s on port %d", cfg.App.Name, cfg.App.Port)
    }),
).Run()
```

### Manual FX Integration

If you need the raw `*config.Config` instance:
    
    fx.Invoke(func(appConfig AppConfig) {
        log.Printf("Starting %s", appConfig.App.Name)
    }),
).Run()
```
### Manual FX Integration

If you need the raw `*config.Config` instance:

```go
fx.New(
    config.Module,
    
    fx.Invoke(func(cfg *config.Config) {
        // Access config directly
        appName := cfg.GetString("app.name")
        port := cfg.GetInt("app.port")
        log.Printf("Starting %s on port %d", appName, port)
    }),
).Run()
```

## Environment Detection

## Advanced Features

### Environment Variable Overrides

## Environment Detection

The config automatically reads from `APP_ENV` environment variable:

```bash
# Development (loads base.yaml + local.yaml)
go run main.go

# Production (loads base.yaml + production.yaml)
export APP_ENV=production
go run main.go

# Staging (loads base.yaml + staging.yaml)
APP_ENV=staging go run main.go
```

Priority:
1. Explicit environment passed to `LoadEnv[T](dir, env)`
2. `APP_ENV` environment variable  
3. Default: `"local" run main.go
```

// Set up hot reload
cfg.OnConfigChange(func() {
    log.Println("Config file changed!")
    
    // Reload your config struct
    var appConfig AppConfig
    if err := cfg.Unmarshal(&appConfig); err != nil {
        log.Printf("Failed to reload config: %v", err)
    }
})

cfg.WatchConfig()
```

### Default Values

```go
cfg, _ := config.New("./config", "local")

// Set defaults
cfg.SetDefault("app.timeout", 30)
cfg.SetDefault("app.retry_count", 3)

### Runtime Value Modification

```go
cfg, _ := config.New("./config", "local")

// Set values at runtime
cfg.Set("app.maintenance_mode", true)
cfg.Set("app.debug_level", "verbose")

// Get the values
maintenanceMode := cfg.GetBool("app.maintenance_mode") // true
```

### Unmarshal Specific Keys

```go
type DatabaseConfig struct {
    Host     string `koanf:"host"`
    Port     int    `koanf:"port"`
    User     string `koanf:"user"`
    Password string `koanf:"password"`
}

cfg, _ := config.New("./config", "local")

var dbConfig DatabaseConfig
if err := cfg.UnmarshalKey("database", &dbConfig); err != nil {
    log.Fatal(err)
}
```

### Check if Key Exists

```go
cfg, _ := config.New("./config", "local")

if cfg.IsSet("feature.experimental_mode") {
    // Feature flag is defined
    enabled := cfg.GetBool("feature.experimental_mode")
}
```

### Access All Settings

```go
cfg, _ := config.New("./config", "local")

// Get all configuration as a map
allSettings := cfg.All()
fmt.Printf("%+v\n", allSettings)
```

## Configuration Priority

Configuration values are loaded and merged in this order (later values override earlier ones):

1. **base.yaml** - Base configuration (required)
2. **{environment}.yaml** - Environment-specific overrides (optional)
3. **APP_* environment variables** - Runtime overrides via env vars
4. **Runtime Set()** - Programmatically set values

Example:

```yaml
# base.yaml
app:
  port: 8080
  name: "MyApp"
  timeout: 30

# production.yaml
app:
  port: 443
  # name and timeout inherited from base.yaml
```

```bash
# Override port via environment variable
export APP_APP_PORT=9000
```

Result:
```go
cfg.GetInt("app.port")       // 9000 (from APP_APP_PORT env var)
cfg.GetString("app.name")    // "MyApp" (from base.yaml)
cfg.GetInt("app.timeout")    // 30
    
    Database struct {
        Host string `koanf:"host" validate:"required"`
        Port int    `koanf:"port" validate:"required"`
    } `koanf:"database"`
}
```

## Testing

The generics API makes testing simple:

```go
func TestMyService(t *testing.T) {)
    require.NoError(t, err)
    
    // Test your service with the config
    svc := NewService(cfg)
    assert.Equal(t, 8080, svc.Port())
}
```

### Testing with Environment Variables

```go
func TestWithEnvOverride(t *testing.T) {
    tmpDir := t.TempDir()
    
    baseConfig := `app:
  port: 8080`
    os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
    
    // Override with env var
    os.Setenv("APP_APP_PORT", "9999")
    defer os.Unsetenv("APP_APP_PORT")
    
    cfg, err := config.Load[AppConfig](tmpDir)
    require.NoError(t, err)
    assert.Equal(t, 9999, cfg.App.Port) // overridden!
    // Minimal config
    os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte("app:\n  name: TestApp"), 0644)
    
    // Provide defaults for everything else
    defaults := AppConfig{}
    defaults.App.Port = 8080
    defaults.Database.Host = "localhost"
    
    cfg, err := config.LoadWithDefaults(tmpDir, "local", defaults)
    require.NoError(t, err)
    assert.Equal(t, "TestApp", cfg.App.Name)
    assert.Equal(t, 8080, cfg.App.Port) // from defaults
}
```

## Error Handling

The package provides specific errors:

```go
cfg, err := config.New("./config", "production")
if err != nil {
    switch {
    case errors.Is(err, config.ErrMissingConfigDir):
        log.Fatal("Config directory not specified")
    case errors.Is(err, config.ErrMissingEnvironment):
        log.Fatal("Environment not specified")
    case errors.Is(err, config.ErrConfigDirNotFound):
        log.Fatal("Config directory does not exist")
    default:
        log.Fatal("Config error:", err)
    }
}
```

## API Reference

### Generic Functions

- `Load[T](configDir) (T, error)` - Load config with auto-detected environment
- `LoadEnv[T](configDir, environment) (T, error)` - Load config with explicit environment
- `MustLoad[T](configDir) T` - Load config or panic

### Config Methods

- `New(configDir, environment) (*Config, error)` - Create new config instance
- `Unmarshal(target) error` - Unmarshal entire config to struct
- `UnmarshalKey(key, target) error` - Unmarshal specific key to struct
- `GetString(key) string` - Get string value
- `GetInt(key) int` - Get int value types:

- `ErrMissingConfigDir` - Config directory not provided
- `ErrConfigDirNotFound` - Config directory doesn't exist
- `ErrUnmarshalFailed` - Failed to unmarshal config to struct

```go
cfg, err := config.Load[AppConfig]("./config")
if err != nil {
    switch {
    case errors.Is(err, config.ErrMissingConfigDir):
        log.Fatal("Config directory not specified")
    case errors.Is(err, config.ErrConfigDirNotFound):
        log.Fatal("Config directory does not exist")
    case errors.Is(err, config.ErrUnmarshalFailed):
        log.Fatal("Failed to parse configstance

### FX Options

- `Module` - FX module for config
- `ProvideConfig[T]() fx.Option` - Provide typed config with generics
- `ProvideConfigDir(dir) fx.Option` - Provide custom config directory
- `ProvideEnvironment(env) fx.Option` - Provide custom environment "github.com/cristiano-pacheco/bricks/pkg/config"
    "go.uber.org/fx"
)context"
    "log"
    
    "github.com/cristiano-pacheco/bricks/pkg/config"
    "go.uber.org/fx"
)

// Application configuration struct
type AppConfig struct {
    App struct {
        Name    string `koanf:"name"`
        Port    int    `koanf:"port"`
        Debug   bool   `koanf:"debug"`
        Timeout int    `koanf:"timeout"`
    } `koanf:"app"`
    
    Database struct {
        Host            string `koanf:"host"`
        Port            int    `koanf:"port"`
        Name            string `koanf:"name"`
        User            string `koanf:"user"`
        Password        string `koanf:"password"`
        MaxConnections  int    `koanf:"max_connections"`
        SSLMode         bool   `koanf:"ssl"`
    } `koanf:"database"`
    
    Redis struct {
        Host     string `koanf:"host"`
        Port     int    `koanf:"port"`
        Password string `koanf:"password"`
        DB       int    `koanf:"db"`
    } `koanf:"redis"`
}

func main() {
    fx.New(
        // Config automatically detects environment from APP_ENV
        config.Module,
        config.ProvideConfig[AppConfig](),
        
        // Use config in your application
        fx.Invoke(runApp),
    ).Run()
}

func runApp(cfg AppConfig, lc fx.Lifecycle) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            log.Printf("Starting %s on port %d (debug: %v)",
                cfg.App.Name,
                cfg.App.Port,
                cfg.App.Debug,
            )
            
            log.Printf("Database: %s@%s:%d/%s (max_conn: %d)",
                cfg.Database.User,
                cfg.Database.Host,
                cfg.Database.Port,
                cfg.Database.Name,
                cfg.Database.MaxConnections,
            )
            
            return nil
        },
    })
}
```

## License

This package is part of the [Bricks](https://github.com/cristiano-pacheco/bricks) framework.       config.ProvideEnvironment(getEnvironment()),
        config.Module,
        
        // Use generics to provide typed config - simple!
        config.ProvideConfig[AppConfig](),
        
        // Inject typed config
        fx.Invoke(runApp),
    ).Run()
}

func getEnvironment() string {
    if env := os.Getenv("APP_ENV"); env != "" {
        return env
    }
    return "local"
}

func runApp(cfg AppConfig, lc fx.Lifecycle) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            log.Printf("Starting %s on port %d (debug: %v)",
                cfg.App.Name,
                cfg.App.Port,
                cfg.App.Debug,
            )
            
            log.Printf("Database: %s@%s:%d/%s (max_conn: %d)",
                cfg.Database.User,
                cfg.Database.Host,
                cfg.Database.Port,
                cfg.Database.Name,
                cfg.Database.MaxConnections,
            )
            
            return nil
        },
    })
}
        
        // Use config in application
        fx.Invoke(runApp),
    ).Run()
}

func getEnvironment() string {
    env := os.Getenv("APP_ENV")
    if env == "" {
        env = "local"
    }
    return env
}

func parseConfig(cfg *config.Config) (AppConfig, error) {
    var appConfig AppConfig
    
    // Set defaults
    cfg.SetDefault("app.timeout", 30)
    cfg.SetDefault("database.max_connections", 25)
    
    // Unmarshal
    if err := cfg.Unmarshal(&appConfig); err != nil {
        return AppConfig{}, fmt.Errorf("failed to parse config: %w", err)
    }
    
    // Validate required fields
    if appConfig.App.Name == "" {
        return AppConfig{}, fmt.Errorf("app.name is required")
    }
    
    return appConfig, nil
}

func runApp(appConfig AppConfig, cfg *config.Config) {
    log.Printf("Starting %s (env: %s)", appConfig.App.Name, cfg.Environment())
    log.Printf("Server port: %d", appConfig.App.Port)
    log.Printf("Database: %s@%s:%d/%s",
        appConfig.Database.User,
        appConfig.Database.Host,
        appConfig.Database.Port,
        appConfig.Database.Name,
    )
    
    // Enable hot reload in development
    if appConfig.App.Debug {
        cfg.OnConfigChange(func() {
            log.Println("⚠️  Config changed! Reload required.")
        })
        cfg.WatchConfig()
}

## License

This package is part of the [Bricks](https://github.com/cristiano-pacheco/bricks) framework.

