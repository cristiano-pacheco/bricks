package logger_test

import (
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_BasicUsage(t *testing.T) {
	// Arrange
	config := logger.DefaultConfig()

	// Act
	log, err := logger.New(config)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, log)
	defer log.Sync()

	log.Info("Test message")
	log.Debug("Debug message", logger.String("key", "value"))
}

func TestLogger_WithOptions(t *testing.T) {
	// Arrange & Act
	log := logger.MustNewWithOptions(
		logger.WithLevel("debug"),
		logger.WithEncoding("console"),
		logger.WithDevelopment(true),
	)

	// Assert
	assert.NotNil(t, log)
	defer log.Sync()

	log.Info("Test with options")
}

func TestLogger_StructuredLogging(t *testing.T) {
	// Arrange
	config := logger.DevelopmentConfig()

	// Act
	log := logger.MustNew(config)

	// Assert
	assert.NotNil(t, log)
	defer log.Sync()

	log.Info("Structured log",
		logger.String("user_id", "123"),
		logger.Int("age", 30),
		logger.Bool("active", true),
	)
}

func TestLogger_WithContext(t *testing.T) {
	// Arrange
	config := logger.DevelopmentConfig()
	log := logger.MustNew(config)
	defer log.Sync()

	// Act
	contextLog := log.With(
		logger.String("request_id", "abc-123"),
		logger.String("user_id", "user-456"),
	)

	// Assert
	assert.NotNil(t, contextLog)

	contextLog.Info("Processing request")
	contextLog.Info("Request completed")
}

func TestLogger_WithError(t *testing.T) {
	// Arrange
	config := logger.DevelopmentConfig()
	log := logger.MustNew(config)
	defer log.Sync()
	err := errors.New("something went wrong")

	// Act
	logWithError := log.WithError(err)

	// Assert
	assert.NotNil(t, logWithError)

	logWithError.Error("Operation failed",
		logger.String("operation", "save_user"),
	)
}
