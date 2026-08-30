package chi

import "errors"

var (
	// ErrInvalidConfig indicates that the server configuration is invalid or incomplete.
	ErrInvalidConfig = errors.New("invalid server configuration")

	// ErrInvalidAddress indicates that a server listen address is invalid.
	ErrInvalidAddress = errors.New("invalid server listen address")

	// ErrInvalidMetricsAddress indicates that a metrics listen address is invalid.
	ErrInvalidMetricsAddress = errors.New("invalid metrics listen address")

	// ErrInvalidPort indicates that the server port is invalid.
	ErrInvalidPort = errors.New("invalid port: must be between 0 and 65535")

	// ErrInvalidMetricsPort indicates that the metrics port is invalid.
	ErrInvalidMetricsPort = errors.New("invalid metrics port: must be between 0 and 65535")

	// ErrPortsEqual indicates that the main port and metrics port cannot be the same.
	ErrPortsEqual = errors.New("metrics port must be different from main server port")

	// ErrListen indicates that a configured listener could not be bound.
	ErrListen = errors.New("failed to bind listener")

	// ErrServerStarted indicates that the server has already been started.
	ErrServerStarted = errors.New("server already started")
)
