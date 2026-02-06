package config_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

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

	// Load config
	cfg, err := config.New(configDir, "local")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.GetString("app.name"))
	fmt.Println(cfg.GetInt("app.port"))
	fmt.Println(cfg.GetBool("app.debug"))

	// Output:
	// MyApp
	// 8080
	// true
}

func Example_unmarshalToStruct() {
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

	cfg, _ := config.New(configDir, "local")

	type AppConfig struct {
		App struct {
			Name string `koanf:"name"`
			Port int    `koanf:"port"`
		} `koanf:"app"`
		Database struct {
			Host string `koanf:"host"`
			Port int    `koanf:"port"`
			User string `koanf:"user"`
		} `koanf:"database"`
	}

	var appConfig AppConfig
	if err := cfg.Unmarshal(&appConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s running on port %d\n", appConfig.App.Name, appConfig.App.Port)
	fmt.Printf("Database: %s@%s:%d\n",
		appConfig.Database.User,
		appConfig.Database.Host,
		appConfig.Database.Port,
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

	// Load production config
	cfg, _ := config.New(configDir, "production")

	fmt.Printf("Name: %s (from base)\n", cfg.GetString("app.name"))
	fmt.Printf("Port: %d (overridden by production)\n", cfg.GetInt("app.port"))
	fmt.Printf("Debug: %v (from base)\n", cfg.GetBool("app.debug"))
	fmt.Printf("Max Connections: %d (only in production)\n", cfg.GetInt("app.max_connections"))

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
	cfg, err := config.Load[AppConfig](configDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s on port %d (debug: %v)\n", cfg.App.Name, cfg.App.Port, cfg.App.Debug)

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
	cfg, err := config.Load[AppConfig](configDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\n", cfg.App.Name)
	fmt.Printf("Port: %d\n", cfg.App.Port)

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

	// MustLoad panics if config cannot be loaded
	// Perfect for critical configuration at startup
	cfg := config.MustLoad[ServerConfig](configDir)

	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)

	// Output:
	// Server: 0.0.0.0:8080
}

func Example_envVarExpansion() {
	tmpDir := os.TempDir()
	configDir := filepath.Join(tmpDir, "example-env-expansion")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	// Set environment variables
	os.Setenv("MY_APP_NAME", "ExpandedApp")
	os.Setenv("DB_HOST", "prod-db.example.com")
	defer func() {
		os.Unsetenv("MY_APP_NAME")
		os.Unsetenv("DB_HOST")
	}()

	// Config with environment variable expansion
	baseConfig := `
app:
  name: ${MY_APP_NAME}
  port: ${APP_PORT:-8080}
  debug: ${DEBUG:-false}

database:
  host: ${DB_HOST}
  port: ${DB_PORT:-5432}
  user: ${DB_USER:-postgres}
`
	os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseConfig), 0644)

	type Config struct {
		App struct {
			Name  string `koanf:"name"`
			Port  int    `koanf:"port"`
			Debug bool   `koanf:"debug"`
		} `koanf:"app"`
		Database struct {
			Host string `koanf:"host"`
			Port int    `koanf:"port"`
			User string `koanf:"user"`
		} `koanf:"database"`
	}

	cfg, _ := config.Load[Config](configDir)

	fmt.Printf("App: %s (port: %d, debug: %v)\n", cfg.App.Name, cfg.App.Port, cfg.App.Debug)
	fmt.Printf("Database: %s@%s:%d\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port)

	// Output:
	// App: ExpandedApp (port: 8080, debug: false)
	// Database: postgres@prod-db.example.com:5432
}
