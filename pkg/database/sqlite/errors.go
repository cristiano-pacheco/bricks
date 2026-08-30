package sqlite

import "errors"

var (
	// ErrInvalidConfig indicates that the SQLite configuration is invalid.
	ErrInvalidConfig = errors.New("invalid sqlite configuration")

	// ErrConnectionFailed indicates that SQLite could not be opened or pinged.
	ErrConnectionFailed = errors.New("failed to connect to sqlite database")

	// ErrMissingDSN indicates that no SQLite path or DSN was provided.
	ErrMissingDSN = errors.New("sqlite database DSN is required")

	// ErrMultipleDSNs indicates that both a path and a DSN were provided.
	ErrMultipleDSNs = errors.New("sqlite database path and DSN cannot both be set")

	// ErrInvalidDSN indicates that the SQLite DSN query could not be parsed.
	ErrInvalidDSN = errors.New("invalid sqlite database DSN")

	// ErrInvalidJournalMode indicates that the SQLite journal mode is unsupported.
	ErrInvalidJournalMode = errors.New("invalid sqlite journal mode")

	// ErrInvalidTxLock indicates that the SQLite transaction lock mode is unsupported.
	ErrInvalidTxLock = errors.New("invalid sqlite transaction lock mode")

	// ErrInvalidBusyTimeout indicates that the SQLite busy timeout is invalid.
	ErrInvalidBusyTimeout = errors.New("invalid sqlite busy timeout")

	// ErrInvalidMaxOpenConnections indicates that SQLite must use one open connection.
	ErrInvalidMaxOpenConnections = errors.New("sqlite must use one open connection")

	// ErrInvalidMaxIdleConnections indicates that SQLite must use at most one idle connection.
	ErrInvalidMaxIdleConnections = errors.New("sqlite must use at most one idle connection")
)
