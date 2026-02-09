package chi

import "errors"

var (
	// ErrInvalidConfig indicates that the server configuration is invalid or incomplete
	ErrInvalidConfig = errors.New("invalid server configuration")

	// ErrInvalidPort indicates that the server port is invalid
	ErrInvalidPort = errors.New("invalid port: must be between 1 and 65535")

	// ErrInvalidMetricsPort indicates that the metrics port is invalid
	ErrInvalidMetricsPort = errors.New("invalid metrics port: must be between 1 and 65535")

	// ErrPortsEqual indicates that the main port and metrics port cannot be the same
	ErrPortsEqual = errors.New("metrics port must be different from main server port")
)
