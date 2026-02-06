package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

// Load loads configuration from YAML files using generics.
// It automatically detects environment from APP_ENV or defaults to "local".
//
// Loading order (later sources override earlier ones):
//  1. base.yaml (required)
//  2. {environment}.yaml (optional, e.g., local.yaml, production.yaml)
//  3. Environment variables with APP_ prefix (e.g., APP_APP_PORT overrides app.port)
//
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
		return result, fmt.Errorf("failed to create config (env=%s): %w", environment, err)
	}
	if err := cfg.Unmarshal(&result); err != nil {
		return result, fmt.Errorf("failed to unmarshal config (env=%s): %w", environment, err)
	}
	return result, nil
}

// MustLoad loads configuration and panics on error.
// Use this for configuration that must be valid for the app to start.
func MustLoad[T any](configDir string) T {
	env := getEnvironment()
	result, err := LoadEnv[T](configDir, env)
	if err != nil {
		panic(fmt.Sprintf("failed to load config from %s (env=%s): %v", configDir, env, err))
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

// expandEnvVars expands environment variables in the format ${VAR} or ${VAR:-default}
// Examples:
//
//	${DATABASE_HOST} -> value of DATABASE_HOST env var
//	${PORT:-8080} -> value of PORT env var, or "8080" if not set
func expandEnvVars(content []byte) []byte {
	// Match ${VAR} or ${VAR:-default}
	re := regexp.MustCompile(`\$\{([A-Z0-9_]+)(:-([^}]*))?\}`)

	expanded := re.ReplaceAllFunc(content, func(match []byte) []byte {
		matches := re.FindSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		varName := string(matches[1])
		defaultValue := ""

		// Check if default value is provided (${VAR:-default})
		if len(matches) >= 4 && matches[3] != nil {
			defaultValue = string(matches[3])
		}

		// Get environment variable value
		if value := os.Getenv(varName); value != "" {
			return []byte(value)
		}

		// Return default value or empty string
		return []byte(defaultValue)
	})

	return expanded
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
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return nil, ErrMissingConfigDir
	}

	environment = strings.ToLower(strings.TrimSpace(environment))
	if environment == "" {
		environment = getEnvironment()
	}

	// Validate config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrConfigDirNotFound, configDir)
	} else if err != nil {
		return nil, fmt.Errorf("failed to access config directory %s: %w", configDir, err)
	}

	cfg := &Config{
		k:           koanf.New("."),
		environment: environment,
		configDir:   configDir,
	}

	// Load base configuration first
	if err := cfg.loadConfigFile("base"); err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Load environment-specific configuration (optional)
	if err := cfg.loadConfigFile(cfg.environment); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load %s.yaml config: %w", cfg.environment, err)
	}

	// Load environment variables with APP_ prefix
	// Example: APP_DATABASE_HOST=localhost overrides database.host
	// The transformation converts: APP_DATABASE_HOST -> database.host
	if err := cfg.k.Load(env.Provider("APP_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "APP_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	return cfg, nil
}

// loadConfigFile loads a specific config file and merges with existing config
// Environment variables in the format ${VAR} or ${VAR:-default} are automatically expanded
func (c *Config) loadConfigFile(name string) error {
	configPath := filepath.Join(c.configDir, name+".yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return err
	}

	// Read file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Expand environment variables (${VAR} or ${VAR:-default})
	expandedContent := expandEnvVars(content)

	// Parse YAML
	parser := yaml.Parser()
	data, err := parser.Unmarshal(expandedContent)
	if err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Merge into koanf
	if err := c.k.Load(&mapProvider{data: data}, nil); err != nil {
		return fmt.Errorf("failed to load config file %s: %w", configPath, err)
	}

	return nil
}

// mapProvider is a simple provider that returns a map
type mapProvider struct {
	data map[string]interface{}
}

func (m *mapProvider) ReadBytes() ([]byte, error) {
	return nil, nil
}

func (m *mapProvider) Read() (map[string]interface{}, error) {
	return m.data, nil
}

// Unmarshal unmarshals the config into a struct
func (c *Config) Unmarshal(target interface{}) error {
	if target == nil {
		return fmt.Errorf("unmarshal target cannot be nil")
	}
	if err := c.k.Unmarshal("", target); err != nil {
		return fmt.Errorf("%w: %v", ErrUnmarshalFailed, err)
	}
	return nil
}

// UnmarshalKey unmarshals a specific key into a struct
func (c *Config) UnmarshalKey(key string, target interface{}) error {
	if target == nil {
		return fmt.Errorf("unmarshal target cannot be nil")
	}
	if !c.k.Exists(key) {
		return fmt.Errorf("config key '%s' not found", key)
	}
	if err := c.k.Unmarshal(key, target); err != nil {
		return fmt.Errorf("%w for key '%s': %v", ErrUnmarshalFailed, key, err)
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
	if !c.k.Exists(key) {
		return make(map[string]interface{})
	}
	// Cut returns a new Koanf instance with the given key as root
	sub := c.k.Cut(key)
	if sub == nil {
		return make(map[string]interface{})
	}
	return sub.All()
}

// IsSet checks if a key is set in the config
func (c *Config) IsSet(key string) bool {
	return c.k.Exists(key)
}

// Set sets a value for the given key at runtime
func (c *Config) Set(key string, value interface{}) {
	c.k.Set(key, value)
}

// All returns all settings as a map
func (c *Config) All() map[string]interface{} {
	return c.k.All()
}

// Environment returns the current environment name
func (c *Config) Environment() string {
	return c.environment
}

// ConfigDir returns the configuration directory
func (c *Config) ConfigDir() string {
	return c.configDir
}
