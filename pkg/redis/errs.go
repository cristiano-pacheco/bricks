package redis

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig indicates that the Redis configuration is invalid or incomplete
	ErrInvalidConfig = errors.New("invalid redis configuration")

	// ErrConnectionFailed indicates that the Redis connection attempt failed
	ErrConnectionFailed = errors.New("failed to connect to redis")

	// ErrMissingURL indicates that the Redis URL is required but not provided
	ErrMissingURL = errors.New("redis URL is required")

	// ErrInvalidClientType indicates that the client type is invalid
	ErrInvalidClientType = errors.New("invalid redis client type")

	// ErrEmptyClientType indicates that the client type is empty
	ErrEmptyClientType = errors.New("redis client type cannot be empty")

	// ErrPingFailed indicates that the ping operation failed
	ErrPingFailed = errors.New("redis ping failed")

	// ErrInvalidDB indicates that the DB number is invalid
	ErrInvalidDB = errors.New("invalid redis DB number")

	// ErrClientClosed indicates that the client is already closed
	ErrClientClosed = errors.New("redis client is closed")
)

// ConnectionError wraps connection errors with additional context
type ConnectionError struct {
	URL        string
	ClientType ClientType
	Attempt    int
	Err        error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection failed to %s (type: %s, attempt %d): %v",
		e.URL, e.ClientType, e.Attempt, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target
func (e *ConnectionError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// ConfigError wraps configuration errors with additional context
type ConfigError struct {
	Field string
	Value interface{}
	Err   error
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("invalid config field '%s' with value '%v': %v",
		e.Field, e.Value, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
