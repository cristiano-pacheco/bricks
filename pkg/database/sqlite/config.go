package sqlite

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	// DefaultBusyTimeout is the time SQLite waits for a locked database.
	DefaultBusyTimeout = 5 * time.Second

	// DefaultJournalMode is the journal mode used by a new configuration.
	DefaultJournalMode = "WAL"

	// DefaultTxLock starts transactions with an immediate write lock.
	DefaultTxLock = "immediate"

	// DefaultMaxOpenConnections limits each process to one SQLite connection.
	DefaultMaxOpenConnections = 1

	// DefaultMaxIdleConnections keeps the single SQLite connection available for reuse.
	DefaultMaxIdleConnections = 1
)

// Config contains the SQLite connection and GORM settings.
type Config struct {
	// DSN is a SQLite path or driver DSN. Path is used when DSN is empty.
	DSN  string `config:"dsn"`
	Path string `config:"path"`

	// SQLite connection settings. Zero values use the Workplan-safe defaults.
	ForeignKeys bool          `config:"foreign_keys"`
	JournalMode string        `config:"journal_mode"`
	BusyTimeout time.Duration `config:"busy_timeout"`
	TxLock      string        `config:"tx_lock"`

	// The connection pool is fixed to one open connection per process.
	MaxOpenConnections int `config:"max_open_connections"`
	MaxIdleConnections int `config:"max_idle_connections"`

	// GORM settings.
	GORMConfig                               *gorm.Config     `config:"gorm_config"`
	PrepareStmt                              bool             `config:"prepare_stmt"`
	PrepareSTMT                              bool             `config:"-"`
	SkipDefaultTransaction                   bool             `config:"skip_default_transaction"`
	DisableForeignKeyConstraintWhenMigrating bool             `config:"disable_foreign_key_constraint_when_migrating"`
	DisableForeignKeyConstraint              bool             `config:"-"`
	EnableLogs                               bool             `config:"enable_logs"`
	Logger                                   logger.Interface `config:"-"`
	NamingStrategy                           schema.Namer     `config:"-"`
}

// DefaultConfig returns a configuration with the SQLite runtime defaults.
func DefaultConfig(dsn string) Config {
	return Config{
		DSN:                dsn,
		ForeignKeys:        true,
		JournalMode:        DefaultJournalMode,
		BusyTimeout:        DefaultBusyTimeout,
		TxLock:             DefaultTxLock,
		MaxOpenConnections: DefaultMaxOpenConnections,
		MaxIdleConnections: DefaultMaxIdleConnections,
	}
}

// Validate checks the configuration before opening SQLite.
func (c Config) Validate() error {
	validators := []func() error{
		c.validateDSN,
		c.validateBusyTimeout,
		c.validateJournalMode,
		c.validateTxLock,
		c.validateMaxOpenConnections,
		c.validateMaxIdleConnections,
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) validateDSN() error {
	hasDSN := strings.TrimSpace(c.DSN) != ""
	hasPath := strings.TrimSpace(c.Path) != ""
	if !hasDSN && !hasPath {
		return ErrMissingDSN
	}
	if hasDSN && hasPath {
		return ErrMultipleDSNs
	}
	return nil
}

func (c Config) validateBusyTimeout() error {
	if c.BusyTimeout < 0 || (c.BusyTimeout > 0 && c.BusyTimeout < time.Millisecond) {
		return fmt.Errorf("%w: %s", ErrInvalidBusyTimeout, c.BusyTimeout)
	}
	return nil
}

func (c Config) validateJournalMode() error {
	if c.JournalMode != "" && !validJournalMode(c.JournalMode) {
		return fmt.Errorf("%w: %q", ErrInvalidJournalMode, c.JournalMode)
	}
	return nil
}

func (c Config) validateTxLock() error {
	if c.TxLock != "" && !validTxLock(c.TxLock) {
		return fmt.Errorf("%w: %q", ErrInvalidTxLock, c.TxLock)
	}
	return nil
}

func (c Config) validateMaxOpenConnections() error {
	if c.MaxOpenConnections < 0 || c.MaxOpenConnections > DefaultMaxOpenConnections {
		return fmt.Errorf("%w: %d", ErrInvalidMaxOpenConnections, c.MaxOpenConnections)
	}
	return nil
}

func (c Config) validateMaxIdleConnections() error {
	if c.MaxIdleConnections < 0 || c.MaxIdleConnections > DefaultMaxIdleConnections {
		return fmt.Errorf("%w: %d", ErrInvalidMaxIdleConnections, c.MaxIdleConnections)
	}
	return nil
}

func (c Config) withDefaults() Config {
	if c.JournalMode == "" {
		c.JournalMode = DefaultJournalMode
	}
	if c.BusyTimeout == 0 {
		c.BusyTimeout = DefaultBusyTimeout
	}
	if c.TxLock == "" {
		c.TxLock = DefaultTxLock
	}
	if c.MaxOpenConnections == 0 {
		c.MaxOpenConnections = DefaultMaxOpenConnections
	}
	if c.MaxIdleConnections == 0 {
		c.MaxIdleConnections = DefaultMaxIdleConnections
	}
	c.ForeignKeys = true
	return c
}

func (c Config) dsn() string {
	if strings.TrimSpace(c.DSN) != "" {
		return c.DSN
	}
	return c.Path
}

func (c Config) connectionDSN() (string, error) {
	dsn := c.dsn()
	base, rawQuery, hasQuery := strings.Cut(dsn, "?")
	values := make(url.Values)
	if hasQuery {
		parsed, err := url.ParseQuery(rawQuery)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrInvalidDSN, err)
		}
		values = parsed
	}

	setQueryValue(values, "_foreign_keys", "on", "_fk")
	setQueryValue(values, "_journal_mode", strings.ToUpper(strings.TrimSpace(c.JournalMode)), "_journal")
	setQueryValue(values, "_busy_timeout", strconv.FormatInt(c.BusyTimeout.Milliseconds(), 10), "_timeout")
	setQueryValue(values, "_txlock", strings.ToLower(strings.TrimSpace(c.TxLock)))

	return base + "?" + values.Encode(), nil
}

func setQueryValue(values url.Values, key, value string, aliases ...string) {
	for _, alias := range append(aliases, key) {
		values.Del(alias)
	}
	values.Set(key, value)
}

func validJournalMode(mode string) bool {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "DELETE", "TRUNCATE", "PERSIST", "MEMORY", "WAL", "OFF":
		return true
	default:
		return false
	}
}

func validTxLock(lock string) bool {
	switch strings.ToLower(strings.TrimSpace(lock)) {
	case "deferred", "immediate", "exclusive":
		return true
	default:
		return false
	}
}
