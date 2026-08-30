package sqlite_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
)

func TestNewConfiguresSQLiteConnection(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "workplan.db")

	// Act
	db, err := sqlite.New(sqlite.Config{DSN: path})

	// Assert
	require.NoError(t, err)
	t.Cleanup(func() { closeDatabase(t, db) })

	var foreignKeys int
	require.NoError(t, db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error)
	assert.Equal(t, 1, foreignKeys)

	var journalMode string
	require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error)
	assert.Equal(t, "wal", strings.ToLower(journalMode))

	var busyTimeout int
	require.NoError(t, db.Raw("PRAGMA busy_timeout").Scan(&busyTimeout).Error)
	assert.Equal(t, int(sqlite.DefaultBusyTimeout/time.Millisecond), busyTimeout)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.Equal(t, sqlite.DefaultMaxOpenConnections, sqlDB.Stats().MaxOpenConnections)
	assert.Equal(t, sqlite.DefaultMaxIdleConnections, sqlDB.Stats().Idle)
}

func TestNewAppliesGORMConfiguration(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "workplan.db")
	gormConfig := &gorm.Config{
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Act
	db, err := sqlite.New(sqlite.Config{DSN: path, GORMConfig: gormConfig})

	// Assert
	require.NoError(t, err)
	t.Cleanup(func() { closeDatabase(t, db) })
	assert.True(t, db.Config.PrepareStmt)
	assert.True(t, db.Config.SkipDefaultTransaction)
	assert.True(t, db.Config.DisableForeignKeyConstraintWhenMigrating)
}

func TestNewUsesImmediateTransactions(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "workplan.db")
	first, err := sqlite.New(sqlite.Config{DSN: path, BusyTimeout: 25 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { closeDatabase(t, first) })

	second, err := sqlite.New(sqlite.Config{DSN: path, BusyTimeout: 25 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { closeDatabase(t, second) })

	transaction := first.Begin()
	require.NoError(t, transaction.Error)
	t.Cleanup(func() { _ = transaction.Rollback().Error })

	// Act
	started := time.Now()
	blocked := second.Begin()
	elapsed := time.Since(started)

	// Assert
	require.Error(t, blocked.Error)
	assert.GreaterOrEqual(t, elapsed, 20*time.Millisecond)
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	// Arrange
	cfg := sqlite.Config{}

	// Act
	_, err := sqlite.New(cfg)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, sqlite.ErrInvalidConfig)
	require.ErrorIs(t, err, sqlite.ErrMissingDSN)
}

func TestNewPreservesSQLiteOpenError(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "missing", "workplan.db")
	cfg := sqlite.Config{DSN: "file:" + path + "?mode=ro"}

	// Act
	_, err := sqlite.New(cfg)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, sqlite.ErrConnectionFailed)
	var driverErr sqlite3.Error
	require.ErrorAs(t, err, &driverErr)
}

func TestModuleClosesDatabaseOnStop(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "workplan.db")
	configDir := t.TempDir()
	t.Setenv("APP_CONFIG_DIR", configDir)
	configFile := filepath.Join(configDir, "base.yaml")
	configData := []byte(fmt.Sprintf(
		"app:\n  sqlite:\n    dsn: %q\n    foreign_keys: true\n    journal_mode: WAL\n    busy_timeout: 25ms\n    tx_lock: immediate\n    max_open_connections: 1\n    max_idle_connections: 1\n",
		path,
	))
	require.NoError(t, os.WriteFile(configFile, configData, 0o600))

	var db *gorm.DB
	app := fx.New(
		sqlite.Module,
		fx.Invoke(func(database *gorm.DB) { db = database }),
		fx.NopLogger,
	)
	ctx := context.Background()

	// Act
	require.NoError(t, app.Start(ctx))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, app.Stop(ctx))
	pingErr := sqlDB.Ping()

	// Assert
	require.Error(t, pingErr)
	assert.Contains(t, pingErr.Error(), "closed")
}

func closeDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}
