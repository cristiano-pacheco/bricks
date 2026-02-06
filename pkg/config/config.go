package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Load loads configuration from YAML files using generics.
// It automatically detects environment from APP_ENV or defaults to "local".
// Loads base.yaml first, then merges environment-specific config.
// The struct T should use `koanf` tags to define field mappings.
//
// Example:
//
//	type AppConfig struct {
//	    App struct {
//	        Name string `koanf:"name"`
//	        Port int    `koanf:"port"`
//	    } `koanf:"app"`
//	}
//
//	cfg, err := config.Load[AppConfig]("./config")
func Load[T any](configDir string) (T, error) {
	environment := getEnvironment()
	return LoadEnv[T](configDir, environment)
}

// LoadEnv loads configuration with explicit environment.
func LoadEnv[T any](configDir, environment string) (T, error) {
	var result T
	cfg, err := New(configDir, environment)
	if err != nil {
		return result, err
	}
	if err := cfg.Unmarshal(&result); err != nil {
		return result, err
	}
	return result, nil
}

// MustLoad loads configuration and panics on error.
// Use this for configuration that must be valid for the app to start.
func MustLoad[T any](configDir string) T {
	result, err := Load[T](configDir)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return result
}

// getEnvironment returns the environment from APP_ENV or defaults to "local"
func getEnvironment() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return strings.ToLower(strings.TrimSpace(env))
	}
	return "local"
}

// Config holds the configuration manager
type Config struct {
	k           *koanf.Koanf
	environment string
	configDir   string
}

// New creates a new configuration manager
// It loads base.yaml first, then merges environment-specific config (e.g., local.yaml)
// configDir: directory containing config files (e.g., "./config")
// environment: environment name (e.g., "local", "staging", "production")
func New(configDir, environment string) (*Config, error) {
	if strings.TrimSpace(configDir) == "" {
		return nil, ErrMissingConfigDir
	}

	if strings.TrimSpace(environment) == "" {
		environment = getEnvironment()
	}

	// Validate config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrConfigDirNotFound, configDir)
	}

	k := koanf.New(".")

	cfg := &Config{
		k:           k,
		environment: strings.ToLower(strings.TrimSpace(environment)),
		configDir:   configDir,
	}

	// Load base configuration first
	if err := cfg.loadConfigFile("base"); err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Load environment-specific configuration (if exists)
	if err := cfg.loadConfigFile(cfg.environment); err != nil {
		// Environment config is optional, only fail if base is missing
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load %s config: %w", cfg.environment, err)
		}
	}

	// Load environment variables with prefix APP_
	// Example: APP_DATABASE_HOST=localhost will override database.host
	cfg.k.Load(env.Provider("APP_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "APP_")), "_", ".", -1)
	}), nil)

	return cfg, nil
}

// loadConfigFile loads a specific config file and merges with existing config
func (c *Config) loadConfigFile(name string) error {
	configPath := filepath.Join(c.configDir, name+".yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return err
	}

	// Load and merge config file
	if err := c.k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return fmt.Errorf("failed to load config file %s: %w", configPath, err)
	}

	return nil
}

// Unmarshal unmarshals the config into a struct
func (c *Config) Unmarshal(target interface{}) error {
	if err := c.k.Unmarshal("", target); err != nil {
		return fmt.Errorf("%w: %v", ErrUnmarshalFailed, err)
	}
	return nil
}

// UnmarshalKey unmarshals a specific key into a struct
func (c *Config) UnmarshalKey(key string, target interface{}) error {
	if err := c.k.Unmarshal(key, target); err != nil {
		return fmt.Errorf("%w for key %s: %v", ErrUnmarshalFailed, key, err)
	}
	return nil
}

// Get returns a value for the given key
func (c *Config) Get(key string) interface{} {
	return c.k.Get(key)
}

// GetString returns a string value for the given key
func (c *Config) GetString(key string) string {
	return c.k.String(key)
}

// GetInt returns an int value for the given key
func (c *Config) GetInt(key string) int {
	return c.k.Int(key)
}

// GetBool returns a bool value for the given key
func (c *Config) GetBool(key string) bool {
	return c.k.Bool(key)
}

// GetFloat64 returns a float64 value for the given key
func (c *Config) GetFloat64(key string) float64 {
	return c.k.Float64(key)
}

// GetStringSlice returns a string slice for the given key
func (c *Config) GetStringSlice(key string) []string {
	return c.k.Strings(key)
}

// GetStringMap returns a map[string]interface{} for the given key
func (c *Config) GetStringMap(key string) map[string]interface{} {
	m := make(map[string]interface{})
	sub := c.k.Cut(key)
	if sub == nil {
		return m
	}
	raw := sub.Raw()
	for k, v := range raw {
		m[k] = v
	}
	return m
}

// IsSet checks if a key is set in the config
func (c *Config) IsSet(key string) bool {
	return c.k.Exists(key)
}

// Set sets a value for the given key at runtime
func (c *Config) Set(key string, value interface{}) {
	c.k.Set(key, value)
}

// AllSettings returns all settings as a map
func (c *Config) AllSettings() map[string]interface{} {
	return c.k.All()
}

// Raw returns all settings as raw map (alias for AllSettings)
func (c *Config) Raw() map[string]interface{} {
	return c.k.Raw()
}

// Environment returns the current environment name
func (c *Config) Environment() string {
	return c.environment
}

// ConfigDir returns the configuration directory
func (c *Config) ConfigDir() string {
	return c.configDir
}

// Koanf returns the underlying koanf instance for advanced usage
func (c *Config) Koanf() *koanf.Koanf {
	return c.k
}
