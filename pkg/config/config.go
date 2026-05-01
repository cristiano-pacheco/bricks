package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
)

const envValuePrefix = "env://"

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
// Config directory is resolved using APP_CONFIG_DIR (default: config).
//
// Example:
//
//	cfg, err := config.New[DatabaseConfig](config.WithPath("app.database"))
func New[T any](options ...Option) (Config[T], error) {
	var result T
	opts := resolveOptions(options)
	environment := getEnvironment()
	configDir, configErr := getConfigDir()
	if configErr != nil {
		return Config[T]{}, configErr
	}
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
	environment = strings.ToLower(strings.TrimSpace(environment))

	k := koanf.New(".")

	// Load base configuration first
	if err := loadConfigFile(k, configDir, "base"); err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Load environment-specific configuration (optional)
	if err := loadConfigFile(k, configDir, environment); err != nil && !errors.Is(err, ErrConfigFileNotFound) {
		return nil, fmt.Errorf("failed to load %s.yaml config: %w", environment, err)
	}

	return k, nil
}

func loadConfigFile(k *koanf.Koanf, configDir, name string) error {
	configPath := filepath.Join(configDir, name+".yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrConfigFileNotFound, configPath)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	data, err := yaml.Parser().Unmarshal(content)
	if err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	resolvedData, err := resolveEnvValues(data)
	if err != nil {
		return fmt.Errorf("failed to resolve env values for config file %s: %w", configPath, err)
	}

	if loadErr := k.Load(&yamlProvider{data: resolvedData}, nil); loadErr != nil {
		return fmt.Errorf("failed to load config file %s: %w", configPath, loadErr)
	}

	return nil
}

func resolveEnvValues(data map[string]interface{}) (map[string]interface{}, error) {
	resolved, err := resolveEnvValue(data)
	if err != nil {
		return nil, err
	}

	resolvedMap, ok := resolved.(map[string]interface{})
	if !ok {
		return nil, errors.New("resolved config data must be a map")
	}

	return resolvedMap, nil
}

func resolveEnvValue(value interface{}) (interface{}, error) {
	switch typed := value.(type) {
	case map[string]interface{}:
		resolved := make(map[string]interface{}, len(typed))
		for key, nestedValue := range typed {
			nextValue, err := resolveEnvValue(nestedValue)
			if err != nil {
				return nil, err
			}
			resolved[key] = nextValue
		}
		return resolved, nil
	case []interface{}:
		resolved := make([]interface{}, len(typed))
		for idx, nestedValue := range typed {
			nextValue, err := resolveEnvValue(nestedValue)
			if err != nil {
				return nil, err
			}
			resolved[idx] = nextValue
		}
		return resolved, nil
	case string:
		if strings.HasPrefix(typed, envValuePrefix) {
			return os.Getenv(strings.TrimPrefix(typed, envValuePrefix)), nil
		}
		return typed, nil
	default:
		return value, nil
	}
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

func getConfigDir() (string, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to resolve root directory: %w", err)
	}

	configDir := strings.TrimSpace(os.Getenv("APP_CONFIG_DIR"))
	if configDir == "" {
		configDir = "config"
	}
	configDir = filepath.Clean(configDir)
	var resolvedPath string
	if filepath.IsAbs(configDir) {
		resolvedPath = configDir
	} else {
		resolvedPath = filepath.Join(rootDir, configDir)
	}
	stat, statErr := os.Stat(resolvedPath) // #nosec G703 -- path is sanitized via filepath.Clean before use
	if os.IsNotExist(statErr) {
		return "", fmt.Errorf("%w: %s", ErrConfigDirNotFound, resolvedPath)
	}
	if statErr != nil {
		return "", fmt.Errorf("failed to access config directory %s: %w", resolvedPath, statErr)
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrConfigDirNotFound, resolvedPath)
	}

	return resolvedPath, nil
}

type yamlProvider struct {
	data map[string]interface{}
}

func (p *yamlProvider) Read() (map[string]interface{}, error) {
	return p.data, nil
}

func (p *yamlProvider) ReadBytes() ([]byte, error) {
	return nil, errors.New("ReadBytes is not supported; use Read instead")
}
