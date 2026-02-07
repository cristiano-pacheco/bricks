package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config[T any] struct {
	value T
}

func (v Config[T]) Get() T {
	return v.value
}

// Option customizes config loading behavior.
type Option func(*loadOptions)

// WithPath unmarshals only a YAML subtree into T (example: "app", "app.database").
func WithPath(path string) Option {
	return func(opts *loadOptions) {
		opts.keyPath = strings.TrimSpace(path)
	}
}

type loadOptions struct {
	keyPath string
}

// New loads and unmarshals configuration into T.
// Environment is always resolved automatically using APP_ENV (default: local).
//
// Example:
//
//	cfg, err := config.New[DatabaseConfig]("./config", config.WithPath("app.database"))
func New[T any](configDir string, options ...Option) (Config[T], error) {
	var result T
	opts := resolveOptions(options)
	environment := getEnvironment()
	k, err := loadKoanf(configDir, environment)
	if err != nil {
		return Config[T]{}, fmt.Errorf("failed to create config (env=%s): %w", environment, err)
	}
	if opts.keyPath != "" {
		if !k.Exists(opts.keyPath) {
			return Config[T]{}, fmt.Errorf("config key '%s' not found", opts.keyPath)
		}
		if unmarshalErr := unmarshalKey(k, opts.keyPath, &result); unmarshalErr != nil {
			return Config[T]{}, fmt.Errorf(
				"failed to unmarshal config key '%s' (env=%s): %w",
				opts.keyPath,
				environment,
				unmarshalErr,
			)
		}
		return Config[T]{value: result}, nil
	}
	if unmarshalErr := unmarshalKey(k, "", &result); unmarshalErr != nil {
		return Config[T]{}, fmt.Errorf("failed to unmarshal config (env=%s): %w", environment, unmarshalErr)
	}
	return Config[T]{value: result}, nil
}

// Internal helpers.
func loadKoanf(configDir, environment string) (*koanf.Koanf, error) {
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

	k := koanf.New(".")

	// Load base configuration first
	if err := loadConfigFile(k, configDir, "base"); err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Load environment-specific configuration (optional)
	if err := loadConfigFile(k, configDir, environment); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load %s.yaml config: %w", environment, err)
	}

	// Load environment variables with APP_ prefix
	// Example: APP_DATABASE_HOST=localhost overrides database.host
	// The transformation converts: APP_DATABASE_HOST -> database.host
	if err := k.Load(env.Provider("APP_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, "APP_")), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	return k, nil
}

func loadConfigFile(k *koanf.Koanf, configDir, name string) error {
	configPath := filepath.Join(configDir, name+".yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	data, err := yaml.Parser().Unmarshal(content)
	if err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	if loadErr := k.Load(&yamlProvider{data: data}, nil); loadErr != nil {
		return fmt.Errorf("failed to load config file %s: %w", configPath, loadErr)
	}

	return nil
}

func unmarshalKey(k *koanf.Koanf, key string, target interface{}) error {
	if target == nil {
		return errors.New("unmarshal target cannot be nil")
	}
	if err := k.UnmarshalWithConf(key, target, koanf.UnmarshalConf{Tag: "config"}); err != nil {
		if key == "" {
			return fmt.Errorf("%w: %w", ErrUnmarshalFailed, err)
		}
		return fmt.Errorf("%w for key '%s': %w", ErrUnmarshalFailed, key, err)
	}
	return nil
}

func resolveOptions(options []Option) loadOptions {
	opts := loadOptions{}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	return opts
}

func getEnvironment() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return strings.ToLower(strings.TrimSpace(env))
	}
	return "local"
}

type yamlProvider struct {
	data map[string]interface{}
}

func (p *yamlProvider) Read() (map[string]interface{}, error) {
	return p.data, nil
}

func (p *yamlProvider) ReadBytes() ([]byte, error) {
	return nil, nil
}
