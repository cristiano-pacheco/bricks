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
	// Create temp config directory
	tmpDir := t.TempDir()

	// Create base.yaml
	baseConfig := `
app:
  name: "TestApp"
  port: 8080
  debug: false

database:
  host: "localhost"
  port: 5432
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create local.yaml
	localConfig := `
app:
  debug: true
  port: 3000
`
	if err := os.WriteFile(filepath.Join(tmpDir, "local.yaml"), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Test loading config
	cfg, err := config.New(tmpDir, "local")
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Verify base values are loaded
	if name := cfg.GetString("app.name"); name != "TestApp" {
		t.Errorf("Expected app.name to be 'TestApp', got '%s'", name)
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
	tmpDir := t.TempDir()

	baseConfig := `
app:
  name: "MyApp"
  port: 8080
  features:
    - "feature1"
    - "feature2"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.New(tmpDir, "local")
	if err != nil {
		t.Fatal(err)
	}

	var appConfig TestConfig
	if err := cfg.Unmarshal(&appConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if appConfig.App.Name != "MyApp" {
		t.Errorf("Expected name to be 'MyApp', got '%s'", appConfig.App.Name)
	}

	if appConfig.App.Port != 8080 {
		t.Errorf("Expected port to be 8080, got %d", appConfig.App.Port)
	}

	if len(appConfig.App.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(appConfig.App.Features))
	}
}

func TestLoad_Generics(t *testing.T) {
	tmpDir := t.TempDir()

	baseConfig := `
app:
  name: "GenericApp"
  port: 9000
  debug: true

database:
  host: "db.example.com"
  port: 5432
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Set environment to test auto-detection
	os.Setenv("APP_ENV", "local")
	defer os.Unsetenv("APP_ENV")

	// Test Load with generics - automatically detects environment
	cfg, err := config.Load[TestConfig](tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config with generics: %v", err)
	}

	if cfg.App.Name != "GenericApp" {
		t.Errorf("Expected app.name to be 'GenericApp', got '%s'", cfg.App.Name)
	}

	if cfg.App.Port != 9000 {
		t.Errorf("Expected app.port to be 9000, got %d", cfg.App.Port)
	}

	if !cfg.App.Debug {
		t.Error("Expected app.debug to be true")
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Expected database.host to be 'db.example.com', got '%s'", cfg.Database.Host)
	}
}

func TestLoadEnv_Explicit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal config
	baseConfig := `
app:
  name: "MinimalApp"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Test LoadEnv with explicit environment
	cfg, err := config.LoadEnv[TestConfig](tmpDir, "local")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Name should come from config file
	if cfg.App.Name != "MinimalApp" {
		t.Errorf("Expected app.name to be 'MinimalApp', got '%s'", cfg.App.Name)
	}
}

func TestMustLoad_Generics(t *testing.T) {
	tmpDir := t.TempDir()

	baseConfig := `
app:
  name: "MustLoadApp"
  port: 3000
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not panic with valid config
	cfg := config.MustLoad[TestConfig](tmpDir)

	if cfg.App.Name != "MustLoadApp" {
		t.Errorf("Expected app.name to be 'MustLoadApp', got '%s'", cfg.App.Name)
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

	baseConfig := `
app:
  name: "EnvTest"
  port: 8080
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Set environment variable to override config
	os.Setenv("APP_APP_PORT", "9999")
	defer os.Unsetenv("APP_APP_PORT")

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
	tmpDir := t.TempDir()

	baseConfig := `
app:
  name: "TestApp"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.New(tmpDir, "local")
	if err != nil {
		t.Fatal(err)
	}

	// Set runtime value
	cfg.Set("runtime.value", "test123")

	if val := cfg.GetString("runtime.value"); val != "test123" {
		t.Errorf("Expected runtime.value to be 'test123', got '%s'", val)
	}
}

func TestIsSet(t *testing.T) {
	tmpDir := t.TempDir()

	baseConfig := `
app:
  name: "TestApp"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.New(tmpDir, "local")
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
