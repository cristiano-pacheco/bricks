package database

import (
	"fmt"
	"math"
	"net"
	"strconv"

	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Config holds the configuration for database connection
type Config struct {
	// Connection parameters
	Host     string
	Port     uint
	Name     string
	User     string
	Password string

	// SSL configuration
	SSLMode     bool
	SSLCert     string
	SSLKey      string
	SSLRootCert string

	// Connection pool settings
	MaxOpenConnections int
	MaxIdleConnections int

	// GORM settings
	PrepareSTMT                 bool
	SkipDefaultTransaction      bool
	DisableForeignKeyConstraint bool

	// Logging
	EnableLogs bool
	Logger     logger.Interface

	// Custom naming strategy
	NamingStrategy schema.Namer

	// Additional parameters
	TimeZone           string
	ApplicationName    string
	SearchPath         string
	StatementTimeout   int // in milliseconds
	LockTimeout        int // in milliseconds
	IdleInTransaction  int // in milliseconds
	ConnectTimeout     int // in seconds
	PreferSimpleProtol bool
}

// Validate validates the database configuration
func (c Config) Validate() error {
	if c.Host == "" {
		return ErrMissingHost
	}
	if c.Name == "" {
		return ErrMissingName
	}
	if c.User == "" {
		return ErrMissingUser
	}
	if c.Port == 0 {
		return ErrMissingPort
	}
	if c.Port > math.MaxUint16 {
		return fmt.Errorf("%w: %d", ErrInvalidPortNumber, c.Port)
	}
	return nil
}

// DSN generates a GORM-compatible DSN string
func (c Config) DSN() string {
	sslMode := "require"
	if !c.SSLMode {
		sslMode = "disable"
	}

	timeZone := c.TimeZone
	if timeZone == "" {
		timeZone = "UTC"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.Host,
		c.User,
		c.Password,
		c.Name,
		c.Port,
		sslMode,
		timeZone,
	)

	// Add optional SSL certificates
	if c.SSLMode && c.SSLCert != "" {
		dsn += fmt.Sprintf(" sslcert=%s", c.SSLCert)
	}
	if c.SSLMode && c.SSLKey != "" {
		dsn += fmt.Sprintf(" sslkey=%s", c.SSLKey)
	}
	if c.SSLMode && c.SSLRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", c.SSLRootCert)
	}

	// Add optional parameters
	if c.ApplicationName != "" {
		dsn += fmt.Sprintf(" application_name=%s", c.ApplicationName)
	}
	if c.SearchPath != "" {
		dsn += fmt.Sprintf(" search_path=%s", c.SearchPath)
	}
	if c.StatementTimeout > 0 {
		dsn += fmt.Sprintf(" statement_timeout=%d", c.StatementTimeout)
	}
	if c.LockTimeout > 0 {
		dsn += fmt.Sprintf(" lock_timeout=%d", c.LockTimeout)
	}
	if c.IdleInTransaction > 0 {
		dsn += fmt.Sprintf(" idle_in_transaction_session_timeout=%d", c.IdleInTransaction)
	}
	if c.ConnectTimeout > 0 {
		dsn += fmt.Sprintf(" connect_timeout=%d", c.ConnectTimeout)
	}

	return dsn
}

// PostgresDSN generates a standard PostgreSQL connection string (postgres://)
func (c Config) PostgresDSN() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}

	sslMode := "require"
	if !c.SSLMode {
		sslMode = "disable"
	}

	timeZone := c.TimeZone
	if timeZone == "" {
		timeZone = "UTC"
	}

	hostPort := net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port)))
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s&TimeZone=%s",
		c.User,
		c.Password,
		hostPort,
		c.Name,
		sslMode,
		timeZone,
	)

	if c.ApplicationName != "" {
		dsn += fmt.Sprintf("&application_name=%s", c.ApplicationName)
	}

	return dsn, nil
}

// Clone creates a deep copy of the configuration
func (c Config) Clone() Config {
	return Config{
		Host:                        c.Host,
		Port:                        c.Port,
		Name:                        c.Name,
		User:                        c.User,
		Password:                    c.Password,
		SSLMode:                     c.SSLMode,
		SSLCert:                     c.SSLCert,
		SSLKey:                      c.SSLKey,
		SSLRootCert:                 c.SSLRootCert,
		MaxOpenConnections:          c.MaxOpenConnections,
		MaxIdleConnections:          c.MaxIdleConnections,
		PrepareSTMT:                 c.PrepareSTMT,
		SkipDefaultTransaction:      c.SkipDefaultTransaction,
		DisableForeignKeyConstraint: c.DisableForeignKeyConstraint,
		EnableLogs:                  c.EnableLogs,
		Logger:                      c.Logger,
		NamingStrategy:              c.NamingStrategy,
		TimeZone:                    c.TimeZone,
		ApplicationName:             c.ApplicationName,
		SearchPath:                  c.SearchPath,
		StatementTimeout:            c.StatementTimeout,
		LockTimeout:                 c.LockTimeout,
		IdleInTransaction:           c.IdleInTransaction,
		ConnectTimeout:              c.ConnectTimeout,
		PreferSimpleProtol:          c.PreferSimpleProtol,
	}
}

// WithDatabase creates a new config with a different database name
func (c Config) WithDatabase(name string) Config {
	cfg := c.Clone()
	cfg.Name = name
	return cfg
}
