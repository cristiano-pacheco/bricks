package database

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/fx"
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

// New creates a new database connection with automatic retry and connection pool configuration.
func New(cfg Config) (*gorm.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	db, err := connect(cfg)
	if err != nil {
		return nil, err
	}

	if errConnectionPool := configureConnectionPool(db, cfg); errConnectionPool != nil {
		return nil, errConnectionPool
	}

	return db, nil
}

// NewWithLifecycle creates a new database connection with fx.Lifecycle management.
// The connection is automatically closed when the application stops.
func NewWithLifecycle(cfg Config, lc fx.Lifecycle) (*gorm.DB, error) {
	db, err := New(cfg)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			sqlDB, errGetSQLDB := db.DB()
			if errGetSQLDB != nil {
				return fmt.Errorf("failed to get underlying sql.DB: %w", errGetSQLDB)
			}
			return sqlDB.Close()
		},
	})

	return db, nil
}

func connect(cfg Config) (*gorm.DB, error) {
	dsn := cfg.DSN()
	gormConfig := buildGormConfig(cfg)
	pgConfig := postgres.Config{DSN: dsn}

	var db *gorm.DB
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()

	for attempt := 1; attempt <= defaultMaxRetries; attempt++ {
		db, err = gorm.Open(postgres.New(pgConfig), gormConfig)
		if err == nil {
			return db, nil
		}

		if attempt < defaultMaxRetries {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: context cancelled: %w", ErrConnectionFailed, ctx.Err())
			case <-time.After(calculateBackoff(attempt, defaultRetryDelay)):
				continue
			}
		}
	}

	return nil, fmt.Errorf("%w: %w (after %d attempts)", ErrConnectionFailed, err, defaultMaxRetries)
}

func configureConnectionPool(db *gorm.DB, cfg Config) error {
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

	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(defaultConnMaxIdleTime)

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

func calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 1 {
		return baseDelay
	}
	multiplier := math.Pow(backoffBase, float64(attempt-1))
	backoff := time.Duration(float64(baseDelay) * multiplier)
	if backoff > maxRetryBackoff {
		backoff = maxRetryBackoff
	}
	return backoff
}
