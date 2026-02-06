package logger_test

import (
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

func Example_basicUsage() {
	// Basic usage with default config
	log := logger.MustNew(logger.DefaultConfig())
	defer log.Sync()

	log.Info("Application started with default config")
}

func Example_developmentConfig() {
	// Development config with colored console output
	log := logger.MustNew(logger.DevelopmentConfig())
	defer log.Sync()

	log.Debug("Debug message", logger.String("environment", "development"))
	log.Info("Info message", logger.Int("port", 8080))
}

func Example_withOptions() {
	// Using options pattern
	log := logger.MustNewWithOptions(
		logger.WithLevel("debug"),
		logger.WithEncoding("console"),
		logger.WithField("service", "example"),
		logger.WithField("version", "1.0.0"),
	)
	defer log.Sync()

	log.Info("User registered",
		logger.String("user_id", "123"),
		logger.String("email", "user@example.com"),
	)
}

func Example_contextLogger() {
	// Child logger with context
	log := logger.MustNew(logger.DevelopmentConfig())
	defer log.Sync()

	reqLog := log.With(
		logger.String("request_id", "req-abc-123"),
		logger.String("ip", "192.168.1.1"),
	)

	reqLog.Info("Processing request")
	reqLog.Info("Request completed", logger.Int("status", 200))
}
