package database

import (
	"context"
	"fmt"
	"math"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	defaultConnectTimeout  = 10 * time.Second
	defaultMaxRetries      = 3
	defaultRetryDelay      = 1 * time.Second
	defaultConnMaxLifetime = 1 * time.Hour
	defaultConnMaxIdleTime = 10 * time.Minute
	maxRetryBackoff        = 30 * time.Second
	backoffBase            = 2
)

// Client encapsulates database operations
type Client struct {
	db     *gorm.DB
	config Config
}

// NewClient creates a new database client with context support
func NewClient(ctx context.Context, cfg Config, opts ...Option) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	db, err := connect(ctx, cfg, options)
	if err != nil {
		return nil, err
	}

	if poolErr := configureConnectionPool(db, cfg, options); poolErr != nil {
		return nil, poolErr
	}

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// DB returns the underlying GORM database instance
func (c *Client) DB() *gorm.DB {
	return c.db
}

// Close closes the database connection
func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// Ping checks if the database connection is alive
func (c *Client) Ping(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Stats returns database statistics
func (c *Client) Stats() (ConnectionStats, error) {
	sqlDB, err := c.db.DB()
	if err != nil {
		return ConnectionStats{}, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	return ConnectionStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

func connect(ctx context.Context, cfg Config, opts options) (*gorm.DB, error) {
	dsn := cfg.DSN()
	gormConfig := buildGormConfig(cfg)
	pgConfig := postgres.Config{DSN: dsn}

	var db *gorm.DB
	var err error

	for attempt := 1; attempt <= opts.MaxRetries; attempt++ {
		db, err = gorm.Open(postgres.New(pgConfig), gormConfig)
		if err == nil {
			return db, nil
		}

		if attempt < opts.MaxRetries {
			if opts.OnRetry != nil {
				opts.OnRetry(attempt, err)
			}

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: context cancelled: %w", ErrConnectionFailed, ctx.Err())
			case <-time.After(calculateBackoff(attempt, opts.RetryDelay)):
				continue
			}
		}
	}

	return nil, fmt.Errorf("%w: %w (after %d attempts)", ErrConnectionFailed, err, opts.MaxRetries)
}

func configureConnectionPool(db *gorm.DB, cfg Config, opts options) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("%w: failed to get underlying sql.DB: %w", ErrConnectionFailed, err)
	}

	if cfg.MaxOpenConnections > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)
	}

	if cfg.MaxIdleConnections > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	}

	connMaxLifetime := opts.ConnMaxLifetime
	if connMaxLifetime == 0 {
		connMaxLifetime = defaultConnMaxLifetime
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	connMaxIdleTime := opts.ConnMaxIdleTime
	if connMaxIdleTime == 0 {
		connMaxIdleTime = defaultConnMaxIdleTime
	}
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	return nil
}

func buildGormConfig(cfg Config) *gorm.Config {
	gormConfig := &gorm.Config{
		PrepareStmt:                              cfg.PrepareSTMT,
		DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKeyConstraint,
		SkipDefaultTransaction:                   cfg.SkipDefaultTransaction,
	}

	if cfg.EnableLogs {
		gormConfig.Logger = cfg.Logger
	}

	if cfg.NamingStrategy != nil {
		gormConfig.NamingStrategy = cfg.NamingStrategy
	}

	return gormConfig
}

// calculateBackoff implements exponential backoff with jitter
func calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 1 {
		return baseDelay
	}
	// Exponential backoff: baseDelay * 2^(attempt-1)
	multiplier := math.Pow(backoffBase, float64(attempt-1))
	backoff := time.Duration(float64(baseDelay) * multiplier)
	// Cap at max retry backoff
	if backoff > maxRetryBackoff {
		backoff = maxRetryBackoff
	}
	return backoff
}
