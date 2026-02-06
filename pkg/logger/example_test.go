package logger_test

import (
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

func Example_basicUsage() {
	// Basic usage with default config
	config := logger.DefaultConfig()
	config.OutputPaths = []string{"/dev/null"}
	config.ErrorOutputPaths = []string{"/dev/null"}
	log := logger.MustNew(config)
	defer log.Sync()

	log.Info("Application started with default config")

	// Output:
}

func Example_developmentConfig() {
	// Development config with colored console output
	config := logger.DevelopmentConfig()
	config.OutputPaths = []string{"/dev/null"}
	config.ErrorOutputPaths = []string{"/dev/null"}
	log := logger.MustNew(config)
	defer log.Sync()

	log.Debug("Debug message", logger.String("environment", "development"))
	log.Info("Info message", logger.Int("port", 8080))

	// Output:
}

func Example_withOptions() {
	// Using options pattern
	log := logger.MustNewWithOptions(
		logger.WithLevel("debug"),
		logger.WithEncoding("console"),
		logger.WithOutputPaths("/dev/null"),
		logger.WithErrorOutputPaths("/dev/null"),
		logger.WithField("service", "example"),
		logger.WithField("version", "1.0.0"),
	)
	defer log.Sync()

	log.Info("User registered",
		logger.String("user_id", "123"),
		logger.String("email", "user@example.com"),
	)

	// Output:
}

func Example_contextLogger() {
	// Child logger with context
	config := logger.DevelopmentConfig()
	config.OutputPaths = []string{"/dev/null"}
	config.ErrorOutputPaths = []string{"/dev/null"}
	log := logger.MustNew(config)
	defer log.Sync()

	reqLog := log.With(
		logger.String("request_id", "req-abc-123"),
		logger.String("ip", "192.168.1.1"),
	)

	reqLog.Info("Processing request")
	reqLog.Info("Request completed", logger.Int("status", 200))

	// Output:
}
