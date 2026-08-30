package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.uber.org/fx"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New opens a GORM database backed by mattn/go-sqlite3.
//
// The connection uses foreign keys, WAL, a five-second busy timeout, immediate
// transactions, and one open connection by default. The caller owns the
// returned connection and must close its underlying sql.DB.
func New(cfg Config) (*gorm.DB, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	dsn, err := cfg.connectionDSN()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	db, err := gorm.Open(gormsqlite.Open(dsn), buildGORMConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("%w: get underlying sql.DB: %w", ErrConnectionFailed, err)
	}
	configureConnectionPool(sqlDB, cfg)

	pingErr := sqlDB.PingContext(context.Background())
	if pingErr != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, pingErr)
	}

	return db, nil
}

// NewWithLifecycle opens a GORM SQLite database and closes it when Fx stops.
// The configuration is loaded by the application's config provider.
func NewWithLifecycle(lc fx.Lifecycle, cfg config.Config[Config]) (*gorm.DB, error) {
	db, err := New(cfg.Get())
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("%w: get underlying sql.DB for lifecycle: %w", ErrConnectionFailed, err)
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			closeErr := sqlDB.Close()
			if closeErr != nil {
				return fmt.Errorf("close sqlite database: %w", closeErr)
			}
			return nil
		},
	})

	return db, nil
}

func configureConnectionPool(sqlDB *sql.DB, cfg Config) {
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
}

func buildGORMConfig(cfg Config) *gorm.Config {
	gormConfig := &gorm.Config{}
	if cfg.GORMConfig != nil {
		*gormConfig = *cfg.GORMConfig
	}

	gormConfig.PrepareStmt = gormConfig.PrepareStmt || cfg.PrepareStmt || cfg.PrepareSTMT
	gormConfig.SkipDefaultTransaction =
		gormConfig.SkipDefaultTransaction || cfg.SkipDefaultTransaction
	gormConfig.DisableForeignKeyConstraintWhenMigrating =
		gormConfig.DisableForeignKeyConstraintWhenMigrating ||
			cfg.DisableForeignKeyConstraintWhenMigrating ||
			cfg.DisableForeignKeyConstraint

	if cfg.Logger != nil {
		gormConfig.Logger = cfg.Logger
	} else if !cfg.EnableLogs && (cfg.GORMConfig == nil || cfg.GORMConfig.Logger == nil) {
		gormConfig.Logger = gormlogger.Discard
	}
	if cfg.NamingStrategy != nil {
		gormConfig.NamingStrategy = cfg.NamingStrategy
	}

	return gormConfig
}
