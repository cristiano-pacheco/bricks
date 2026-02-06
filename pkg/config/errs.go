package config

import "errors"

var (
	// ErrMissingConfigDir indicates that the config directory is required but not provided
	ErrMissingConfigDir = errors.New("config directory is required")

	// ErrMissingEnvironment indicates that the environment is required but not provided
	ErrMissingEnvironment = errors.New("environment is required")

	// ErrConfigDirNotFound indicates that the config directory does not exist
	ErrConfigDirNotFound = errors.New("config directory not found")

	// ErrConfigFileNotFound indicates that a required config file was not found
	ErrConfigFileNotFound = errors.New("config file not found")

	// ErrUnmarshalFailed indicates that unmarshaling config to struct failed
	ErrUnmarshalFailed = errors.New("failed to unmarshal config")
)
