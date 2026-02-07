package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

type TestConfig struct {
	App struct {
		Name     string   `koanf:"name"`
		Port     int      `koanf:"port"`
		Debug    bool     `koanf:"debug"`
		Features []string `koanf:"features"`
	} `koanf:"app"`
	Database struct {
		Host string `koanf:"host"`
		Port int    `koanf:"port"`
	} `koanf:"database"`
}

func TestNew(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	// Test loading config with local environment
	cfg, err := config.New(configDir, "local")
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Verify base values are loaded
	if name := cfg.GetString("app.name"); name != "MyApp" {
		t.Errorf("Expected app.name to be 'MyApp', got '%s'", name)
	}

	// Verify local override works
	if debug := cfg.GetBool("app.debug"); !debug {
		t.Error("Expected app.debug to be true (overridden by local.yaml)")
	}

	if port := cfg.GetInt("app.port"); port != 3000 {
		t.Errorf("Expected app.port to be 3000 (overridden by local.yaml), got %d", port)
	}

	// Verify base value not overridden
	if dbHost := cfg.GetString("database.host"); dbHost != "localhost" {
		t.Errorf("Expected database.host to be 'localhost', got '%s'", dbHost)
	}
}

func TestUnmarshal(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	cfg, err := config.New(configDir, "local")
	if err != nil {
		t.Fatal(err)
	}

	var appConfig TestConfig
	if unmarshalErr := cfg.Unmarshal(&appConfig); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal config: %v", unmarshalErr)
	}

	if appConfig.App.Name != "MyApp" {
		t.Errorf("Expected name to be 'MyApp', got '%s'", appConfig.App.Name)
	}

	// Port is overridden by local.yaml (3000)
	if appConfig.App.Port != 3000 {
		t.Errorf("Expected port to be 3000, got %d", appConfig.App.Port)
	}

	if len(appConfig.App.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(appConfig.App.Features))
	}
}

func TestLoad_Generics(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	// Set environment to test auto-detection
	t.Setenv("APP_ENV", "local")

	// Test Load with generics - automatically detects environment
	cfg, err := config.Load[TestConfig](configDir)
	if err != nil {
		t.Fatalf("Failed to load config with generics: %v", err)
	}

	if cfg.App.Name != "MyApp" {
		t.Errorf("Expected app.name to be 'MyApp', got '%s'", cfg.App.Name)
	}

	// Port is overridden by local.yaml (3000)
	if cfg.App.Port != 3000 {
		t.Errorf("Expected app.port to be 3000, got %d", cfg.App.Port)
	}

	if !cfg.App.Debug {
		t.Error("Expected app.debug to be true")
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected database.host to be 'localhost', got '%s'", cfg.Database.Host)
	}
}

func TestLoadEnv_Explicit(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	// Test LoadEnv with explicit production environment
	cfg, err := config.LoadEnv[TestConfig](configDir, "production")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Name should come from production.yaml
	if cfg.App.Name != "ProductionApp" {
		t.Errorf("Expected app.name to be 'ProductionApp', got '%s'", cfg.App.Name)
	}

	// Port should be overridden by production.yaml
	if cfg.App.Port != 443 {
		t.Errorf("Expected app.port to be 443, got %d", cfg.App.Port)
	}

	// Debug should be false in production
	if cfg.App.Debug {
		t.Error("Expected app.debug to be false in production")
	}
}

func TestMustLoad_Generics(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	// Should not panic with valid config
	cfg := config.MustLoad[TestConfig](configDir)

	if cfg.App.Name != "MyApp" {
		t.Errorf("Expected app.name to be 'MyApp', got '%s'", cfg.App.Name)
	}
}

func TestMustLoad_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustLoad to panic with invalid config")
		}
	}()

	// This should panic because directory doesn't exist
	_ = config.MustLoad[TestConfig]("/nonexistent/path")
}

func TestEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755); err != nil {
		t.Fatal(err)
	}

	baseConfig := `
app:
  name: "EnvTest"
  port: 8080
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Set environment variable to override config
	t.Setenv("APP_APP_PORT", "9999")

	cfg, err := config.LoadEnv[TestConfig](tmpDir, "local")
	if err != nil {
		t.Fatal(err)
	}

	// Port should be overridden by environment variable
	if cfg.App.Port != 9999 {
		t.Errorf("Expected app.port to be 9999 (from env var), got %d", cfg.App.Port)
	}
}

func TestMissingBaseConfig(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	// Only create local.yaml, no base.yaml
	localConfig := `
app:
  name: "TestApp"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "local.yaml"), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := config.New(tmpDir, "local")
	if err == nil {
		t.Error("Expected error when base.yaml is missing")
	}
}

func TestMissingConfigDir(t *testing.T) {
	_, err := config.New("", "local")
	if err == nil {
		t.Error("Expected error when config dir is empty")
	}
}

func TestConfigDirNotFound(t *testing.T) {
	_, err := config.New("/nonexistent/path", "local")
	if err == nil {
		t.Error("Expected error when config directory doesn't exist")
	}
}

func TestSetAndGet(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	cfg, err := config.New(configDir, "local")
	if err != nil {
		t.Fatal(err)
	}

	// Set runtime value
	if setErr := cfg.Set("runtime.value", "test123"); setErr != nil {
		t.Fatalf("Failed to set runtime value: %v", setErr)
	}

	if val := cfg.GetString("runtime.value"); val != "test123" {
		t.Errorf("Expected runtime.value to be 'test123', got '%s'", val)
	}
}

func TestIsSet(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	cfg, err := config.New(configDir, "local")
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.IsSet("app.name") {
		t.Error("Expected app.name to be set")
	}

	if cfg.IsSet("app.nonexistent") {
		t.Error("Expected app.nonexistent to not be set")
	}
}

func TestMultipleEnvironments(t *testing.T) {
	// Use real config files from pkg/config/config directory
	configDir := "./config"

	tests := []struct {
		name          string
		environment   string
		expectedApp   string
		expectedPort  int
		expectedDebug bool
	}{
		{
			name:          "Local environment",
			environment:   "local",
			expectedApp:   "MyApp",
			expectedPort:  3000,
			expectedDebug: true,
		},
		{
			name:          "Production environment",
			environment:   "production",
			expectedApp:   "ProductionApp",
			expectedPort:  443,
			expectedDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.LoadEnv[TestConfig](configDir, tt.environment)
			if err != nil {
				t.Fatalf("Failed to load config for %s: %v", tt.environment, err)
			}

			if cfg.App.Name != tt.expectedApp {
				t.Errorf("Expected app.name to be '%s', got '%s'", tt.expectedApp, cfg.App.Name)
			}

			if cfg.App.Port != tt.expectedPort {
				t.Errorf("Expected app.port to be %d, got %d", tt.expectedPort, cfg.App.Port)
			}

			if cfg.App.Debug != tt.expectedDebug {
				t.Errorf("Expected app.debug to be %v, got %v", tt.expectedDebug, cfg.App.Debug)
			}
		})
	}
}
