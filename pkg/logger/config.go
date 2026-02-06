package logger

import (
	"go.uber.org/zap/zapcore"
)

// Config holds the logger configuration
type Config struct {
	// Level sets the logging level (debug, info, warn, error, dpanic, panic, fatal)
	Level string

	// Encoding sets the log encoding format ("json" or "console")
	Encoding string

	// Development enables development mode (stacktraces on warnings, more verbose)
	Development bool

	// DisableCaller disables automatic caller field
	DisableCaller bool

	// DisableStacktrace disables automatic stacktrace capturing
	DisableStacktrace bool

	// OutputPaths defines where logs are written (e.g., ["stdout", "/var/log/app.log"])
	OutputPaths []string

	// ErrorOutputPaths defines where internal logger errors are written
	ErrorOutputPaths []string

	// InitialFields adds initial fields to every log entry
	InitialFields map[string]interface{}

	// SamplingConfig enables sampling to reduce log volume in production
	SamplingConfig *SamplingConfig
}

// SamplingConfig configures log sampling
type SamplingConfig struct {
	// Initial sets the number of log entries to keep per second
	Initial int

	// Thereafter only logs 1 in N entries after Initial is exceeded
	Thereafter int
}

// DefaultConfig returns a production-ready configuration
func DefaultConfig() Config {
	return Config{
		Level:             "info",
		Encoding:          "json",
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		InitialFields:     make(map[string]interface{}),
	}
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() Config {
	return Config{
		Level:             "debug",
		Encoding:          "console",
		Development:       true,
		DisableCaller:     false,
		DisableStacktrace: false,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		InitialFields:     make(map[string]interface{}),
	}
}

// parseLevel converts string level to zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
