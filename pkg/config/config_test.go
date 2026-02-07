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
		Name     string   `config:"name"`
		Port     int      `config:"port"`
		Debug    bool     `config:"debug"`
		Features []string `config:"features"`
	} `config:"app"`
	Database struct {
		Host string `config:"host"`
		Port int    `config:"port"`
	} `config:"database"`
}

func TestNew(t *testing.T) {
	t.Run("should load config successfully", func(t *testing.T) {
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
	})
}

func TestUnmarshal(t *testing.T) {
	t.Run("should unmarshal config into struct", func(t *testing.T) {
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
	})
}

func TestLoad_Generics(t *testing.T) {
	t.Run("should load config using generics", func(t *testing.T) {
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
	})
}

func TestLoadEnv_Explicit(t *testing.T) {
	t.Run("should load production environment explicitly", func(t *testing.T) {
		// Arrange
		configDir := "./config"

		// Act
		cfg, err := config.LoadEnv[TestConfig](configDir, "production")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "ProductionApp", cfg.App.Name)
		assert.Equal(t, 443, cfg.App.Port)
		assert.False(t, cfg.App.Debug)
	})
}

func TestMustLoad_Generics(t *testing.T) {
	t.Run("should load config without error", func(t *testing.T) {
		// Arrange
		configDir := "./config"

		// Act
		cfg := config.MustLoad[TestConfig](configDir)

		// Assert
		assert.Equal(t, "MyApp", cfg.App.Name)
	})
}

func TestMustLoad_Panic(t *testing.T) {
	t.Run("should panic on invalid path", func(t *testing.T) {
		// Arrange
		invalidPath := "/nonexistent/path"

		// Act & Assert
		assert.Panics(t, func() {
			_ = config.MustLoad[TestConfig](invalidPath)
		})
	})
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("should override config with environment variables", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)
		require.NoError(t, err)

		baseConfig := `
app:
  name: "EnvTest"
  port: 8080
`
		err = os.WriteFile(
			filepath.Join(tmpDir, "base.yaml"),
			[]byte(baseConfig),
			0644,
		)
		require.NoError(t, err)
		t.Setenv("APP_APP_PORT", "9999")

		// Act
		cfg, err := config.LoadEnv[TestConfig](tmpDir, "local")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 9999, cfg.App.Port)
	})
}

func TestMissingBaseConfig(t *testing.T) {
	t.Run("should return error when base.yaml is missing", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)
		require.NoError(t, err)

		localConfig := `
app:
  name: "TestApp"
`
		err = os.WriteFile(
			filepath.Join(tmpDir, "local.yaml"),
			[]byte(localConfig),
			0644,
		)
		require.NoError(t, err)

		// Act
		_, err = config.New(tmpDir, "local")

		// Assert
		require.Error(t, err)
	})
}

func TestMissingConfigDir(t *testing.T) {
	t.Run("should return error when config dir is empty", func(t *testing.T) {
		// Arrange
		emptyDir := ""

		// Act
		_, err := config.New(emptyDir, "local")

		// Assert
		require.Error(t, err)
	})
}

func TestConfigDirNotFound(t *testing.T) {
	t.Run("should return error when config dir does not exist", func(t *testing.T) {
		// Arrange
		nonexistentPath := "/nonexistent/path"

		// Act
		_, err := config.New(nonexistentPath, "local")

		// Assert
		require.Error(t, err)
	})
}

func TestSetAndGet(t *testing.T) {
	t.Run("should set and get runtime value", func(t *testing.T) {
		// Arrange
		configDir := "./config"
		cfg, err := config.New(configDir, "local")
		require.NoError(t, err)

		// Act
		setErr := cfg.Set("runtime.value", "test123")

		// Assert
		require.NoError(t, setErr)
		assert.Equal(t, "test123", cfg.GetString("runtime.value"))
	})
}

func TestIsSet(t *testing.T) {
	t.Run("should check if config key exists", func(t *testing.T) {
		// Arrange
		configDir := "./config"
		cfg, err := config.New(configDir, "local")
		require.NoError(t, err)

		// Act & Assert
		assert.True(t, cfg.IsSet("app.name"))
		assert.False(t, cfg.IsSet("app.nonexistent"))
	})
}

func TestMultipleEnvironments(t *testing.T) {
	configDir := "./config"

	tests := []struct {
		name          string
		environment   string
		expectedApp   string
		expectedPort  int
		expectedDebug bool
	}{
		{
			name:          "should load local environment",
			environment:   "local",
			expectedApp:   "MyApp",
			expectedPort:  3000,
			expectedDebug: true,
		},
		{
			name:          "should load production environment",
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

func TestCustomLoad(t *testing.T) {
	type DatabaseConfig struct {
		Host string `config:"host"`
		Port int    `config:"port"`
		Name string `config:"name"`
		User string `config:"user"`
	}

	type AppSection struct {
		Name     string   `config:"name"`
		Port     int      `config:"port"`
		Debug    bool     `config:"debug"`
		Features []string `config:"features"`
	}

	t.Run("should load database section with CustomLoad", func(t *testing.T) {
		// Arrange
		configDir := "./config"
		t.Setenv("APP_ENV", "local")

		// Act
		cfg, err := config.CustomLoad[DatabaseConfig](configDir, "database")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 5432, cfg.Port)
		assert.Equal(t, "myapp_db", cfg.Name)
		assert.Equal(t, "postgres", cfg.User)
	})

	t.Run("should load app section with CustomLoad", func(t *testing.T) {
		// Arrange
		configDir := "./config"
		t.Setenv("APP_ENV", "local")

		// Act
		cfg, err := config.CustomLoad[AppSection](configDir, "app")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "MyApp", cfg.Name)
		assert.Equal(t, 3000, cfg.Port)
		assert.True(t, cfg.Debug)
		assert.Equal(t, []string{"api", "web", "admin"}, cfg.Features)
	})

	t.Run("should load database section with CustomLoadEnv", func(t *testing.T) {
		// Arrange
		configDir := "./config"

		// Act
		cfg, err := config.CustomLoadEnv[DatabaseConfig](configDir, "production", "database")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "db.production.com", cfg.Host)
		assert.Equal(t, 5432, cfg.Port)
	})

	t.Run("should return error when key path does not exist", func(t *testing.T) {
		// Arrange
		configDir := "./config"

		// Act
		_, err := config.CustomLoad[DatabaseConfig](configDir, "nonexistent.key")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent.key")
	})

	t.Run("should panic with MustCustomLoad on invalid key", func(t *testing.T) {
		// Arrange
		configDir := "./config"

		// Act & Assert
		assert.Panics(t, func() {
			config.MustCustomLoad[DatabaseConfig](configDir, "nonexistent.key")
		})
	})

	t.Run("should load successfully with MustCustomLoad", func(t *testing.T) {
		// Arrange
		configDir := "./config"
		t.Setenv("APP_ENV", "local")

		// Act & Assert
		assert.NotPanics(t, func() {
			cfg := config.MustCustomLoad[DatabaseConfig](configDir, "database")
			assert.Equal(t, "localhost", cfg.Host)
		})
	})
}
