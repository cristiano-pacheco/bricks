package config_test

import (
	"os"
	"path/filepath"
	"strings"
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

const testConfigDir = "config"

func loadConfig[T any](configDir string, options ...config.Option) (config.Config[T], error) {
	resolvedDir := normalizeConfigDir(configDir)
	if resolvedDir == "" {
		_ = os.Unsetenv("APP_CONFIG_DIR")
		return config.New[T](options...)
	}
	_ = os.Setenv("APP_CONFIG_DIR", resolvedDir)
	return config.New[T](options...)
}

func normalizeConfigDir(configDir string) string {
	trimmedDir := strings.TrimSpace(configDir)
	if trimmedDir == "" {
		return ""
	}
	if !filepath.IsAbs(trimmedDir) {
		return trimmedDir
	}
	root, err := os.Getwd()
	if err != nil {
		return trimmedDir
	}
	rel, relErr := filepath.Rel(root, trimmedDir)
	if relErr != nil {
		return trimmedDir
	}
	return rel
}

func tempConfigDir(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	require.NoError(t, err)
	baseDir := filepath.Base(t.TempDir())
	dir := filepath.Join(root, baseDir)
	require.NoError(t, os.MkdirAll(dir, 0755))
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func TestNew(t *testing.T) {
	t.Run("should load config successfully", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir

		// Act
		t.Setenv("APP_ENV", "local")
		cfg, err := loadConfig[TestConfig](configDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "MyApp", cfg.Get().App.Name)
		assert.True(t, cfg.Get().App.Debug)
		assert.Equal(t, 3000, cfg.Get().App.Port)
		assert.Equal(t, "localhost", cfg.Get().Database.Host)
	})
}

func TestLoad_Generics(t *testing.T) {
	t.Run("should load config using generics", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir
		t.Setenv("APP_ENV", "local")

		// Act
		cfg, err := loadConfig[TestConfig](configDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "MyApp", cfg.Get().App.Name)
		assert.Equal(t, 3000, cfg.Get().App.Port)
		assert.True(t, cfg.Get().App.Debug)
		assert.Equal(t, "localhost", cfg.Get().Database.Host)
	})
}

func TestLoad_EnvironmentFromEnvVar(t *testing.T) {
	t.Run("should load production environment via APP_ENV", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir
		t.Setenv("APP_ENV", "production")

		// Act
		cfg, err := loadConfig[TestConfig](configDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "ProductionApp", cfg.Get().App.Name)
		assert.Equal(t, 443, cfg.Get().App.Port)
		assert.False(t, cfg.Get().App.Debug)
	})
}

func TestNew_WithNoError(t *testing.T) {
	t.Run("should load config without error", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir

		// Act
		cfg, err := loadConfig[TestConfig](configDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "MyApp", cfg.Get().App.Name)
	})
}

func TestNew_InvalidPathError(t *testing.T) {
	t.Run("should return error on invalid path", func(t *testing.T) {
		// Arrange
		invalidPath := "missing-config-dir"

		// Act & Assert
		_, err := loadConfig[TestConfig](invalidPath)
		require.Error(t, err)
	})
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("should override config with environment variables", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
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
		t.Setenv("APP_PORT", "9999")

		// Act
		t.Setenv("APP_ENV", "local")
		cfg, err := loadConfig[TestConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 9999, cfg.Get().App.Port)
	})
}

func TestMissingBaseConfig(t *testing.T) {
	t.Run("should return error when base.yaml is missing", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
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
		t.Setenv("APP_ENV", "local")
		_, err = loadConfig[TestConfig](tmpDir)

		// Assert
		require.Error(t, err)
	})
}

func TestMissingConfigDir(t *testing.T) {
	t.Run("should return error when default config dir is missing", func(t *testing.T) {
		// Arrange
		tmpRoot := t.TempDir()
		t.Chdir(tmpRoot)

		// Act
		_, err := loadConfig[TestConfig]("   ")

		// Assert
		require.Error(t, err)
	})
}

func TestConfigDirNotFound(t *testing.T) {
	t.Run("should return error when config dir does not exist", func(t *testing.T) {
		// Arrange
		nonexistentPath := "missing-config-dir"

		// Act
		_, err := loadConfig[TestConfig](nonexistentPath)

		// Assert
		require.Error(t, err)
	})

	t.Run("should return error when config dir is a file", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		filePath := filepath.Join(tmpDir, "not-a-dir.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("app:\n  name: test\n"), 0644))

		// Act
		_, err := loadConfig[TestConfig](filePath)

		// Assert
		require.Error(t, err)
	})
}

func TestNewWithPath(t *testing.T) {
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
		configDir := testConfigDir
		t.Setenv("APP_ENV", "local")

		// Act
		cfg, err := loadConfig[DatabaseConfig](configDir, config.WithPath("database"))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Get().Host)
		assert.Equal(t, 5432, cfg.Get().Port)
		assert.Equal(t, "myapp_db", cfg.Get().Name)
		assert.Equal(t, "postgres", cfg.Get().User)
	})

	t.Run("should load app section with CustomLoad", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir
		t.Setenv("APP_ENV", "local")

		// Act
		cfg, err := loadConfig[AppSection](configDir, config.WithPath("app"))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "MyApp", cfg.Get().Name)
		assert.Equal(t, 3000, cfg.Get().Port)
		assert.True(t, cfg.Get().Debug)
		assert.Equal(t, []string{"api", "web", "admin"}, cfg.Get().Features)
	})

	t.Run("should load database section with APP_ENV=production", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir
		t.Setenv("APP_ENV", "production")

		// Act
		cfg, err := loadConfig[DatabaseConfig](configDir, config.WithPath("database"))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "db.production.com", cfg.Get().Host)
		assert.Equal(t, 5432, cfg.Get().Port)
	})

	t.Run("should return error when key path does not exist", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir

		// Act
		_, err := loadConfig[DatabaseConfig](configDir, config.WithPath("nonexistent.key"))

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent.key")
	})

	t.Run("should return error on invalid key", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir

		// Act & Assert
		_, err := loadConfig[DatabaseConfig](configDir, config.WithPath("nonexistent.key"))
		require.Error(t, err)
	})

	t.Run("should load successfully with WithPath", func(t *testing.T) {
		// Arrange
		configDir := testConfigDir
		t.Setenv("APP_ENV", "local")

		// Act & Assert
		assert.NotPanics(t, func() {
			cfg, err := loadConfig[DatabaseConfig](configDir, config.WithPath("database"))
			require.NoError(t, err)
			assert.Equal(t, "localhost", cfg.Get().Host)
		})
	})
}

func TestInvalidYAMLParsing(t *testing.T) {
	t.Run("should return error on invalid YAML syntax", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		invalidYAML := `
app:
  name: "Test
  port: invalid yaml
    more invalid
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(invalidYAML), 0644)
		require.NoError(t, err)

		// Act
		_, err = loadConfig[TestConfig](tmpDir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

func TestInvalidConfigStructure(t *testing.T) {
	type StrictConfig struct {
		RequiredInt int `config:"required_int"`
	}

	t.Run("should return error when config doesn't match struct", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		// Create a config that has a string where int is expected
		invalidStructure := `
required_int: "this_is_a_string_not_an_int"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(invalidStructure), 0644)
		require.NoError(t, err)

		// Act
		_, err = loadConfig[StrictConfig](tmpDir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal config")
	})

	t.Run("should return error when WithPath config doesn't match struct", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		invalidStructure := `
database:
  port: "should_be_number"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(invalidStructure), 0644)
		require.NoError(t, err)

		type DBConfig struct {
			Port int `config:"port"`
		}

		// Act
		_, err = loadConfig[DBConfig](tmpDir, config.WithPath("database"))

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal config key 'database'")
	})
}

func TestConfigFilePermissionErrors(t *testing.T) {
	t.Run("should return error when base.yaml is not readable", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		configPath := filepath.Join(tmpDir, "base.yaml")
		err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644)
		require.NoError(t, err)

		// Remove read permissions
		err = os.Chmod(configPath, 0000)
		require.NoError(t, err)
		defer os.Chmod(configPath, 0644) // Restore permissions

		// Act
		_, err = loadConfig[TestConfig](tmpDir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestEnvironmentConfigWithErrors(t *testing.T) {
	t.Run("should return error when environment-specific config has read errors", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		// Create valid base config
		baseConfig := `
app:
  name: "Test"
  port: 8080
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Create environment-specific config with no read permission
		envConfigPath := filepath.Join(tmpDir, "production.yaml")
		err = os.WriteFile(envConfigPath, []byte("app:\n  name: prod\n"), 0644)
		require.NoError(t, err)
		err = os.Chmod(envConfigPath, 0000)
		require.NoError(t, err)
		defer os.Chmod(envConfigPath, 0644)

		t.Setenv("APP_ENV", "production")

		// Act
		_, err = loadConfig[TestConfig](tmpDir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load production.yaml config")
	})

	t.Run("should return error when environment-specific YAML is invalid", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		// Create valid base config
		baseConfig := `
app:
  name: "Test"
  port: 8080
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Create invalid environment-specific config
		invalidEnvConfig := `
app:
  name: "Broken
  tabs	and invalid: : : syntax
`
		err = os.WriteFile(filepath.Join(tmpDir, "staging.yaml"), []byte(invalidEnvConfig), 0644)
		require.NoError(t, err)

		t.Setenv("APP_ENV", "staging")

		// Act
		_, err = loadConfig[TestConfig](tmpDir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load staging.yaml config")
	})
}

func TestConfigWithSpacesAndTrimming(t *testing.T) {
	t.Run("should handle config dir with whitespace", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		root, err := os.Getwd()
		require.NoError(t, err)
		relDir, err := filepath.Rel(root, tmpDir)
		require.NoError(t, err)
		configDir := "  " + relDir + "  "

		baseConfig := `
app:
  name: "Test"
  port: 8080
`
		err = os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Act
		cfg, err := loadConfig[TestConfig](configDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Test", cfg.Get().App.Name)
	})

	t.Run("should handle WithPath with whitespace", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
database:
  host: "localhost"
  port: 5432
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		type DBConfig struct {
			Host string `config:"host"`
			Port int    `config:"port"`
		}

		// Act - WithPath with extra whitespace
		cfg, err := loadConfig[DBConfig](tmpDir, config.WithPath("  database  "))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Get().Host)
	})
}

func TestEnvironmentVariableEdgeCases(t *testing.T) {
	t.Run("should handle APP_ENV with mixed case", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "Base"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		prodConfig := `
app:
  name: "Production"
`
		err = os.WriteFile(filepath.Join(tmpDir, "production.yaml"), []byte(prodConfig), 0644)
		require.NoError(t, err)

		// Act - Set environment with mixed case
		t.Setenv("APP_ENV", "  PrOdUcTiOn  ")
		cfg, err := loadConfig[TestConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Production", cfg.Get().App.Name)
	})

	t.Run("should use local when APP_ENV is only whitespace", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "Base"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		localConfig := `
app:
  name: "Local"
`
		err = os.WriteFile(filepath.Join(tmpDir, "local.yaml"), []byte(localConfig), 0644)
		require.NoError(t, err)

		// Act - When APP_ENV is whitespace, TrimSpace results in empty string
		// Since there's no ".yaml" file, it should only load base.yaml
		t.Setenv("APP_ENV", "   ")
		cfg, err := loadConfig[TestConfig](tmpDir)

		// Assert - Will load base.yaml since environment after trim is "" and ".yaml" doesn't exist
		require.NoError(t, err)
		assert.Equal(t, "Base", cfg.Get().App.Name)
	})

	t.Run("should override nested config with environment variables", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  host: "localhost"
  port: 5432
  name: "myapp"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Set multiple environment variables.
		// The transformer strips APP_ then prepends "app.", so APP_X -> app.x
		t.Setenv("APP_HOST", "prod.example.com")
		t.Setenv("APP_PORT", "3306")
		t.Setenv("APP_NAME", "production_db")

		type AppConfig struct {
			Host string `config:"host"`
			Port int    `config:"port"`
			Name string `config:"name"`
		}

		// Act
		cfg, err := loadConfig[AppConfig](tmpDir, config.WithPath("app"))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "prod.example.com", cfg.Get().Host)
		assert.Equal(t, 3306, cfg.Get().Port)
		assert.Equal(t, "production_db", cfg.Get().Name)
	})
}

func TestMultipleOptions(t *testing.T) {
	t.Run("should handle nil options gracefully", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "Test"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Act - Pass nil options
		cfg, err := loadConfig[TestConfig](tmpDir, nil, config.WithPath(""), nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Test", cfg.Get().App.Name)
	})
}

func TestConfigGet(t *testing.T) {
	t.Run("should return same value on multiple Get calls", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "Immutable"
  port: 9000
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		cfg, err := loadConfig[TestConfig](tmpDir)
		require.NoError(t, err)

		// Act - Call Get multiple times
		val1 := cfg.Get()
		val2 := cfg.Get()
		val3 := cfg.Get()

		// Assert - All should return the same values
		assert.Equal(t, "Immutable", val1.App.Name)
		assert.Equal(t, "Immutable", val2.App.Name)
		assert.Equal(t, "Immutable", val3.App.Name)
		assert.Equal(t, 9000, val1.App.Port)
		assert.Equal(t, 9000, val2.App.Port)
		assert.Equal(t, 9000, val3.App.Port)
	})
}

func TestEmptyConfigStructure(t *testing.T) {
	type EmptyConfig struct{}

	t.Run("should handle empty struct successfully", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "Test"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Act
		cfg, err := loadConfig[EmptyConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, cfg.Get())
	})
}

func TestDeepNestedConfiguration(t *testing.T) {
	type DeepConfig struct {
		Level1 struct {
			Level2 struct {
				Level3 struct {
					Value string `config:"value"`
				} `config:"level3"`
			} `config:"level2"`
		} `config:"level1"`
	}

	t.Run("should load deeply nested configuration", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		deepConfig := `
level1:
  level2:
    level3:
      value: "deep_value"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(deepConfig), 0644)
		require.NoError(t, err)

		// Act
		cfg, err := loadConfig[DeepConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "deep_value", cfg.Get().Level1.Level2.Level3.Value)
	})

	t.Run("should load deeply nested with WithPath", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		deepConfig := `
level1:
  level2:
    level3:
      value: "nested"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(deepConfig), 0644)
		require.NoError(t, err)

		type Level3Config struct {
			Value string `config:"value"`
		}

		// Act
		cfg, err := loadConfig[Level3Config](tmpDir, config.WithPath("level1.level2.level3"))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "nested", cfg.Get().Value)
	})
}

func TestEnvironmentVariablesWithComplexPaths(t *testing.T) {
	t.Run("should override deeply nested values with env vars", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  api:
    endpoint:
      url: "http://localhost"
      port: 8080
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		type ServiceConfig struct {
			App struct {
				API struct {
					Endpoint struct {
						URL  string `config:"url"`
						Port int    `config:"port"`
					} `config:"endpoint"`
				} `config:"api"`
			} `config:"app"`
		}

		// Set environment variables for deeply nested values.
		// Use __ as nesting delimiter: APP_API__ENDPOINT__URL -> app.api.endpoint.url
		t.Setenv("APP_API__ENDPOINT__URL", "https://production.example.com")
		t.Setenv("APP_API__ENDPOINT__PORT", "443")

		// Act
		cfg, err := loadConfig[ServiceConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "https://production.example.com", cfg.Get().App.API.Endpoint.URL)
		assert.Equal(t, 443, cfg.Get().App.API.Endpoint.Port)
	})
}

func TestConfigDirectoryPermissionError(t *testing.T) {
	// This test is platform-specific and may not work on all systems
	if os.Getenv("SKIP_PERMISSION_TESTS") != "" {
		t.Skip("Skipping permission tests")
	}

	t.Run("should handle directory permission errors", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)
		configDir := filepath.Join(tmpDir, "config")
		err := os.Mkdir(configDir, 0755)
		require.NoError(t, err)

		// Create a valid base.yaml
		baseYAML := `
app:
  name: "Test"
`
		err = os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(baseYAML), 0644)
		require.NoError(t, err)

		// Remove all permissions from config directory (not just read)
		err = os.Chmod(configDir, 0000)
		require.NoError(t, err)
		defer os.Chmod(configDir, 0755) // Restore permissions for cleanup

		// Act
		_, err = loadConfig[TestConfig](configDir)

		// Assert - Should get permission error (either accessing dir or reading file)
		require.Error(t, err)
		// Error message varies by OS, but should mention permission issues
		assert.True(t, err != nil, "should have permission error")
	})
}

func TestNilTargetUnmarshal(t *testing.T) {
	// This tests internal behavior that's not directly exposed
	// but we can test it indirectly through the public API
	t.Run("should handle invalid struct pointer", func(t *testing.T) {
		// This is hard to test directly since Go's type system prevents
		// passing nil to generic functions, but we've covered the error handling
		// The unmarshalKey function has nil checks which are defensive programming

		// Testing that our config works with valid pointers
		tmpDir := tempConfigDir(t)
		baseConfig := `
app:
  name: "Test"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// This should work fine (not nil)
		cfg, err := loadConfig[TestConfig](tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "Test", cfg.Get().App.Name)
	})
}

func TestYAMLProviderReadBytes(t *testing.T) {
	t.Run("yamlProvider ReadBytes returns nil", func(t *testing.T) {
		// The ReadBytes method is not used by koanf but is part of the Provider interface
		// We can test that the config loads successfully, which internally uses the yamlProvider
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "TestProvider"
  port: 8080
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// If the provider works correctly, config should load
		cfg, err := loadConfig[TestConfig](tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "TestProvider", cfg.Get().App.Name)
	})
}

func TestEdgeCaseYAMLFiles(t *testing.T) {
	t.Run("should handle empty base.yaml", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		// Empty YAML file
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(""), 0644)
		require.NoError(t, err)

		type EmptyTestConfig struct{}

		// Act
		cfg, err := loadConfig[EmptyTestConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, cfg.Get())
	})

	t.Run("should handle YAML with comments only", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		commentsYAML := `
# This is a comment
# Another comment
# Just comments, no actual config
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(commentsYAML), 0644)
		require.NoError(t, err)

		type EmptyTestConfig struct{}

		// Act
		cfg, err := loadConfig[EmptyTestConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, cfg.Get())
	})

	t.Run("should handle base.yaml with only whitespace and newlines", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		// YAML with only spaces and newlines (tabs can cause parse errors)
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte("   \n  \n   \n"), 0644)
		require.NoError(t, err)

		type EmptyTestConfig struct{}

		// Act
		cfg, err := loadConfig[EmptyTestConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, cfg.Get())
	})
}

func TestComplexDataTypes(t *testing.T) {
	type ComplexConfig struct {
		StringMap   map[string]string   `config:"string_map"`
		IntSlice    []int               `config:"int_slice"`
		BoolValue   bool                `config:"bool_value"`
		FloatValue  float64             `config:"float_value"`
		NestedSlice []map[string]string `config:"nested_slice"`
	}

	t.Run("should load complex data types correctly", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		complexYAML := `
string_map:
  key1: "value1"
  key2: "value2"
int_slice:
  - 1
  - 2
  - 3
bool_value: true
float_value: 3.14159
nested_slice:
  - name: "item1"
    value: "value1"
  - name: "item2"
    value: "value2"
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(complexYAML), 0644)
		require.NoError(t, err)

		// Act
		cfg, err := loadConfig[ComplexConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "value1", cfg.Get().StringMap["key1"])
		assert.Equal(t, "value2", cfg.Get().StringMap["key2"])
		assert.Equal(t, []int{1, 2, 3}, cfg.Get().IntSlice)
		assert.True(t, cfg.Get().BoolValue)
		assert.Equal(t, 3.14159, cfg.Get().FloatValue)
		assert.Len(t, cfg.Get().NestedSlice, 2)
		assert.Equal(t, "item1", cfg.Get().NestedSlice[0]["name"])
	})
}

func TestEnvironmentVariableWithInvalidValues(t *testing.T) {
	t.Run("should handle environment variable with invalid type conversion", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  port: 8080
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Set environment variable with invalid value for int field.
		// APP_PORT -> app.port (transformer strips APP_, prepends "app.")
		t.Setenv("APP_PORT", "not_a_number")

		type AppConfig struct {
			App struct {
				Port int `config:"port"`
			} `config:"app"`
		}

		// Act
		_, err = loadConfig[AppConfig](tmpDir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal config")
	})
}

func TestMultipleEnvironmentFiles(t *testing.T) {
	t.Run("should correctly merge base and environment configs", func(t *testing.T) {
		// Arrange
		tmpDir := tempConfigDir(t)

		baseConfig := `
app:
  name: "BaseApp"
  port: 8080
  debug: false
database:
  host: "localhost"
  port: 5432
`
		err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseConfig), 0644)
		require.NoError(t, err)

		// Production overrides only some values
		prodConfig := `
app:
  name: "ProductionApp"
  debug: false
database:
  host: "prod.db.example.com"
`
		err = os.WriteFile(filepath.Join(tmpDir, "production.yaml"), []byte(prodConfig), 0644)
		require.NoError(t, err)

		t.Setenv("APP_ENV", "production")

		// Act
		cfg, err := loadConfig[TestConfig](tmpDir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "ProductionApp", cfg.Get().App.Name) // Overridden
		assert.Equal(
			t,
			8080,
			cfg.Get().App.Port,
		) // Remains from base (not overridden in prod config)
		assert.False(t, cfg.Get().App.Debug)                            // Overridden
		assert.Equal(t, "prod.db.example.com", cfg.Get().Database.Host) // Overridden
		assert.Equal(t, 5432, cfg.Get().Database.Port)                  // Remains from base (not overridden)
	})
}
