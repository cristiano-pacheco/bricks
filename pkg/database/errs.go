package database

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig indicates that the database configuration is invalid or incomplete
	ErrInvalidConfig = errors.New("invalid database configuration")

	// ErrConnectionFailed indicates that the database connection attempt failed
	ErrConnectionFailed = errors.New("failed to connect to database")

	// ErrInvalidPortNumber indicates that the port number exceeds the valid range
	ErrInvalidPortNumber = errors.New("port number exceeds valid range")

	// ErrMissingHost indicates that the database host is required but not provided
	ErrMissingHost = errors.New("database host is required")

	// ErrMissingName indicates that the database name is required but not provided
	ErrMissingName = errors.New("database name is required")

	// ErrMissingUser indicates that the database user is required but not provided
	ErrMissingUser = errors.New("database user is required")

	// ErrMissingPort indicates that the database port is required but not provided
	ErrMissingPort = errors.New("database port is required")
)

// ConnectionError wraps connection errors with additional context
type ConnectionError struct {
	Host    string
	Port    uint
	DB      string
	Attempt int
	Err     error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection failed to %s:%d/%s (attempt %d): %v",
		e.Host, e.Port, e.DB, e.Attempt, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target
func (e *ConnectionError) Is(target error) bool {
	return errors.Is(e.Err, target)
}
