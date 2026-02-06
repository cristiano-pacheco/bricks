package logger_test

import (
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

func TestLogger_BasicUsage(t *testing.T) {
	// Test with default config
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Sync()

	log.Info("Test message")
	log.Debug("Debug message", logger.String("key", "value"))
}

func TestLogger_WithOptions(t *testing.T) {
	log := logger.MustNewWithOptions(
		logger.WithLevel("debug"),
		logger.WithEncoding("console"),
		logger.WithDevelopment(true),
	)
	defer log.Sync()

	log.Info("Test with options")
}

func TestLogger_StructuredLogging(t *testing.T) {
	log := logger.MustNew(logger.DevelopmentConfig())
	defer log.Sync()

	log.Info("Structured log",
		logger.String("user_id", "123"),
		logger.Int("age", 30),
		logger.Bool("active", true),
	)
}

func TestLogger_WithContext(t *testing.T) {
	log := logger.MustNew(logger.DevelopmentConfig())
	defer log.Sync()

	// Create child logger with context
	contextLog := log.With(
		logger.String("request_id", "abc-123"),
		logger.String("user_id", "user-456"),
	)

	contextLog.Info("Processing request")
	contextLog.Info("Request completed")
}

func TestLogger_WithError(t *testing.T) {
	log := logger.MustNew(logger.DevelopmentConfig())
	defer log.Sync()

	err := errors.New("something went wrong")
	log.WithError(err).Error("Operation failed",
		logger.String("operation", "save_user"),
	)
}
