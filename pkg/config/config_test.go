package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// Arrange
	configDir := "./config"

	// Act
	cfg, err := config.New(configDir, "local")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "MyApp", cfg.GetString("app.name"))
	assert.True(t, cfg.GetBool("app.debug"))
	assert.Equal(t, 3000, cfg.GetInt("app.port"))
	assert.Equal(t, "localhost", cfg.GetString("database.host"))
}

func TestUnmarshal(t *testing.T) {
	// Arrange
	configDir := "./config"
	cfg, err := config.New(configDir, "local")
	require.NoError(t, err)

	var appConfig TestConfig

	// Act
	unmarshalErr := cfg.Unmarshal(&appConfig)

	// Assert
	require.NoError(t, unmarshalErr)
	assert.Equal(t, "MyApp", appConfig.App.Name)
	assert.Equal(t, 3000, appConfig.App.Port)
	assert.Len(t, appConfig.App.Features, 3)
}

func TestLoad_Generics(t *testing.T) {
	// Arrange
	configDir := "./config"
	t.Setenv("APP_ENV", "local")

	// Act
	cfg, err := config.Load[TestConfig](configDir)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "MyApp", cfg.App.Name)
	assert.Equal(t, 3000, cfg.App.Port)
	assert.True(t, cfg.App.Debug)
	assert.Equal(t, "localhost", cfg.Database.Host)
}

func TestLoadEnv_Explicit(t *testing.T) {
	// Arrange
	configDir := "./config"

	// Act
	cfg, err := config.LoadEnv[TestConfig](configDir, "production")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ProductionApp", cfg.App.Name)
	assert.Equal(t, 443, cfg.App.Port)
	assert.False(t, cfg.App.Debug)
}

func TestMustLoad_Generics(t *testing.T) {
	// Arrange
	configDir := "./config"

	// Act
	cfg := config.MustLoad[TestConfig](configDir)

	// Assert
	assert.Equal(t, "MyApp", cfg.App.Name)
}

func TestMustLoad_Panic(t *testing.T) {
	// Arrange & Act & Assert
	assert.Panics(t, func() {
		_ = config.MustLoad[TestConfig]("/nonexistent/path")
	})
}

func TestEnvironmentVariables(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)
	require.NoError(t, err)

	baseConfig := `
app:
  name: "EnvTest"
  port: 8080
`
	err = os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
	require.NoError(t, err)
	t.Setenv("APP_APP_PORT", "9999")

	// Act
	cfg, err := config.LoadEnv[TestConfig](tmpDir, "local")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 9999, cfg.App.Port)
}

func TestMissingBaseConfig(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)
	require.NoError(t, err)

	localConfig := `
app:
  name: "TestApp"
`
	err = os.WriteFile(filepath.Join(tmpDir, "local.yaml"), []byte(localConfig), 0644)
	require.NoError(t, err)

	// Act
	_, err = config.New(tmpDir, "local")

	// Assert
	require.Error(t, err)
}

func TestMissingConfigDir(t *testing.T) {
	// Arrange & Act
	_, err := config.New("", "local")

	// Assert
	require.Error(t, err)
}

func TestConfigDirNotFound(t *testing.T) {
	// Arrange & Act
	_, err := config.New("/nonexistent/path", "local")

	// Assert
	require.Error(t, err)
}

func TestSetAndGet(t *testing.T) {
	// Arrange
	configDir := "./config"
	cfg, err := config.New(configDir, "local")
	require.NoError(t, err)

	// Act
	setErr := cfg.Set("runtime.value", "test123")

	// Assert
	require.NoError(t, setErr)
	assert.Equal(t, "test123", cfg.GetString("runtime.value"))
}

func TestIsSet(t *testing.T) {
	// Arrange
	configDir := "./config"
	cfg, err := config.New(configDir, "local")
	require.NoError(t, err)

	// Act & Assert
	assert.True(t, cfg.IsSet("app.name"))
	assert.False(t, cfg.IsSet("app.nonexistent"))
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
			// Arrange & Act
			cfg, err := config.LoadEnv[TestConfig](configDir, tt.environment)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedApp, cfg.App.Name)
			assert.Equal(t, tt.expectedPort, cfg.App.Port)
			assert.Equal(t, tt.expectedDebug, cfg.App.Debug)
		})
	}
}
