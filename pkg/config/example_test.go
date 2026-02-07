package config_test

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

func exampleLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func Example_basicUsage() {
	// Setup temp directory for example
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-config")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	// Create base.yaml
	baseConfig := `
app:
  name: "MyApp"
  port: 8080
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	// Create local.yaml
	localConfig := `
app:
  debug: true
`
	os.WriteFile(filepath.Join(configDir, "local.yaml"), []byte(localConfig), 0644)

	type AppConfig struct {
		App struct {
			Name  string `config:"name"`
			Port  int    `config:"port"`
			Debug bool   `config:"debug"`
		} `config:"app"`
	}

	os.Setenv("APP_ENV", "local")
	defer os.Unsetenv("APP_ENV")
	cfg, err := config.New[AppConfig](configDir)
	if err != nil {
		exampleLogger().Error("failed to load config", "err", err)
		return
	}

	fmt.Println(cfg.Get().App.Name)
	fmt.Println(cfg.Get().App.Port)
	fmt.Println(cfg.Get().App.Debug)

	// Output:
	// MyApp
	// 8080
	// true
}

func Example_loadToStruct() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-struct")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	baseConfig := `
app:
  name: "MyApp"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "postgres"
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	type AppConfig struct {
		App struct {
			Name string `config:"name"`
			Port int    `config:"port"`
		} `config:"app"`
		Database struct {
			Host string `config:"host"`
			Port int    `config:"port"`
			User string `config:"user"`
		} `config:"database"`
	}

	os.Setenv("APP_ENV", "local")
	defer os.Unsetenv("APP_ENV")
	appConfig, err := config.New[AppConfig](configDir)
	if err != nil {
		exampleLogger().Error("failed to load config", "err", err)
		return
	}

	fmt.Printf("%s running on port %d\n", appConfig.Get().App.Name, appConfig.Get().App.Port)
	fmt.Printf("Database: %s@%s:%d\n",
		appConfig.Get().Database.User,
		appConfig.Get().Database.Host,
		appConfig.Get().Database.Port,
	)

	// Output:
	// MyApp running on port 8080
	// Database: postgres@localhost:5432
}

func Example_multipleEnvironments() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-env")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	// Base config
	baseConfig := `
app:
  name: "MyApp"
  port: 8080
  debug: false
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	// Production overrides
	prodConfig := `
app:
  port: 443
  max_connections: 100
`
	os.WriteFile(filepath.Join(configDir, "production.yaml"), []byte(prodConfig), 0644)

	type AppConfig struct {
		App struct {
			Name           string `config:"name"`
			Port           int    `config:"port"`
			Debug          bool   `config:"debug"`
			MaxConnections int    `config:"max_connections"`
		} `config:"app"`
	}

	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")
	cfg, err := config.New[AppConfig](configDir)
	if err != nil {
		exampleLogger().Error("failed to load config", "err", err)
		return
	}

	fmt.Printf("Name: %s (from base)\n", cfg.Get().App.Name)
	fmt.Printf("Port: %d (overridden by production)\n", cfg.Get().App.Port)
	fmt.Printf("Debug: %v (from base)\n", cfg.Get().App.Debug)
	fmt.Printf("Max Connections: %d (only in production)\n", cfg.Get().App.MaxConnections)

	// Output:
	// Name: MyApp (from base)
	// Port: 443 (overridden by production)
	// Debug: false (from base)
	// Max Connections: 100 (only in production)
}

func Example_loadWithGenerics() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-generics")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	baseConfig := `
app:
  name: "GenericApp"
  port: 9000
  debug: true
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	// Define your config struct
	type AppConfig struct {
		App struct {
			Name  string `koanf:"name"`
			Port  int    `koanf:"port"`
			Debug bool   `koanf:"debug"`
		} `koanf:"app"`
	}

	// Load config using generics - simple and type-safe!
	// Environment is automatically detected from APP_ENV
	cfg, err := config.New[AppConfig](configDir)
	if err != nil {
		exampleLogger().Error("failed to load config", "err", err)
		return
	}

	fmt.Printf("%s on port %d (debug: %v)\n", cfg.Get().App.Name, cfg.Get().App.Port, cfg.Get().App.Debug)

	// Output:
	// GenericApp on port 9000 (debug: true)
}

func Example_loadWithDefaults() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-defaults")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	// Minimal config file
	baseConfig := `
app:
  name: "MinimalApp"
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	type AppConfig struct {
		App struct {
			Name    string `koanf:"name"`
			Port    int    `koanf:"port"`
			Timeout int    `koanf:"timeout"`
		} `koanf:"app"`
	}

	// With Koanf, you can set defaults in your struct or use environment variables
	// Example: APP_APP_PORT=8080 will set app.port
	cfg, err := config.New[AppConfig](configDir)
	if err != nil {
		exampleLogger().Error("failed to load config", "err", err)
		return
	}

	fmt.Printf("Name: %s\n", cfg.Get().App.Name)
	fmt.Printf("Port: %d\n", cfg.Get().App.Port)

	// Output:
	// Name: MinimalApp
	// Port: 0
}

func Example_mustLoad() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-must")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	baseConfig := `
server:
  host: "0.0.0.0"
  port: 8080
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	type ServerConfig struct {
		Server struct {
			Host string `koanf:"host"`
			Port int    `koanf:"port"`
		} `koanf:"server"`
	}

	cfg, err := config.New[ServerConfig](configDir)
	if err != nil {
		exampleLogger().Error("failed to load server config", "err", err)
		return
	}

	fmt.Printf("Server: %s:%d\n", cfg.Get().Server.Host, cfg.Get().Server.Port)

	// Output:
	// Server: 0.0.0.0:8080
}

func Example_customLoad() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-custom-load")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	baseConfig := `
app:
  name: "MyApp"
  port: 8080
  database:
    host: "localhost"
    port: 5432
    name: "myapp_db"
    user: "postgres"
redis:
  host: "localhost"
  port: 6379
  db: 0
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	// Define a struct for only the database section
	type DatabaseConfig struct {
		Host string `config:"host"`
		Port int    `config:"port"`
		Name string `config:"name"`
		User string `config:"user"`
	}

	// Define a struct for only the redis section
	type RedisConfig struct {
		Host string `config:"host"`
		Port int    `config:"port"`
		DB   int    `config:"db"`
	}

	// Load only the database section using WithPath
	dbCfg, err := config.New[DatabaseConfig](configDir, config.WithPath("app.database"))
	if err != nil {
		exampleLogger().Error("failed to load database config", "err", err)
		return
	}

	fmt.Printf("Database: %s@%s:%d/%s\n", dbCfg.Get().User, dbCfg.Get().Host, dbCfg.Get().Port, dbCfg.Get().Name)

	// Load only the redis section
	redisCfg, err := config.New[RedisConfig](configDir, config.WithPath("redis"))
	if err != nil {
		exampleLogger().Error("failed to load redis config", "err", err)
		return
	}

	fmt.Printf("Redis: %s:%d (DB %d)\n", redisCfg.Get().Host, redisCfg.Get().Port, redisCfg.Get().DB)

	// Output:
	// Database: postgres@localhost:5432/myapp_db
	// Redis: localhost:6379 (DB 0)
}
