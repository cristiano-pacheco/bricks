package logger

import (
	"context"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for logging operations
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// Info logs an info message
	Info(msg string, fields ...Field)

	// Warn logs a warning message
	Warn(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, fields ...Field)

	// DPanic logs a panic message in development, error in production
	DPanic(msg string, fields ...Field)

	// Panic logs a panic message and panics
	Panic(msg string, fields ...Field)

	// Fatal logs a fatal message and calls os.Exit(1)
	Fatal(msg string, fields ...Field)

	// With creates a child logger with additional fields
	With(fields ...Field) Logger

	// WithError adds an error field to the logger
	WithError(err error) Logger

	// Sync flushes any buffered log entries
	Sync() error

	// GetZapLogger returns the underlying zap.Logger (for advanced usage)
	GetZapLogger() *zap.Logger
}

// Field is an alias for zap.Field
type Field = zap.Field

// ZapLogger implements Logger interface
type ZapLogger struct {
	logger *zap.Logger
}

var _ Logger = (*ZapLogger)(nil) // Ensure ZapLogger implements Logger

// New creates a new logger instance from config
func New(config Config) (*ZapLogger, error) {
	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(parseLevel(config.Level)),
		Development:       config.Development,
		DisableCaller:     config.DisableCaller,
		DisableStacktrace: config.DisableStacktrace,
		Encoding:          config.Encoding,
		EncoderConfig:     getEncoderConfig(config.Encoding),
		OutputPaths:       config.OutputPaths,
		ErrorOutputPaths:  config.ErrorOutputPaths,
		InitialFields:     config.InitialFields,
	}

	if config.SamplingConfig != nil {
		zapConfig.Sampling = &zap.SamplingConfig{
			Initial:    config.SamplingConfig.Initial,
			Thereafter: config.SamplingConfig.Thereafter,
		}
	}

	logger, err := zapConfig.Build(
		zap.AddCallerSkip(1), // Skip one level to show actual caller
	)
	if err != nil {
		return nil, err
	}

	return &ZapLogger{logger: logger}, nil
}

func NewWithLifecycle(lc fx.Lifecycle, cfg config.Config[Config]) (*ZapLogger, error) {
	// Create the logger
	log, err := New(cfg.Get())
	if err != nil {
		return nil, err
	}

	// Register lifecycle hook to sync logger on shutdown
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return log.Sync()
		},
	})

	return log, nil
}

// MustNew creates a new logger or panics on error
func MustNew(config Config) *ZapLogger {
	logger, err := New(config)
	if err != nil {
		panic(err)
	}
	return logger
}

// getEncoderConfig returns encoder configuration
func getEncoderConfig(encoding string) zapcore.EncoderConfig {
	if encoding == "console" {
		return zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	// JSON encoding (production)
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fields...)
}

func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fields...)
}

func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fields...)
}

func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fields...)
}

func (l *ZapLogger) DPanic(msg string, fields ...Field) {
	l.logger.DPanic(msg, fields...)
}

func (l *ZapLogger) Panic(msg string, fields ...Field) {
	l.logger.Panic(msg, fields...)
}

func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *ZapLogger) With(fields ...Field) Logger {
	return &ZapLogger{logger: l.logger.With(fields...)}
}

func (l *ZapLogger) WithError(err error) Logger {
	return &ZapLogger{logger: l.logger.With(zap.Error(err))}
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

func (l *ZapLogger) GetZapLogger() *zap.Logger {
	return l.logger
}

// Common field helpers

// String creates a string field
func String(key, val string) Field {
	return zap.String(key, val)
}

// Int creates an int field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 creates an int64 field
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Float64 creates a float64 field
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool creates a bool field
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Error creates an error field
func Error(err error) Field {
	return zap.Error(err)
}

// Any creates a field from any value
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Duration creates a duration field
func Duration(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Time creates a time field
func Time(key string, val interface{}) Field {
	return zap.Any(key, val)
}
