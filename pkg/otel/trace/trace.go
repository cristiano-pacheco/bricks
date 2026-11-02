package trace

import (
	"context"

	gotrace "github.com/cristiano-pacheco/go-otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Initialize configures the global tracer.
func Initialize(config TracerConfig) error {
	return gotrace.Initialize(config.otelConfig())
}

// MustInitialize initializes the global tracer and panics if it fails.
func MustInitialize(config TracerConfig) {
	gotrace.MustInitialize(config.otelConfig())
}

// Span starts a new span with the given name and options.
func Span(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return gotrace.Span(ctx, name, opts...)
}

// Shutdown gracefully shuts down the tracer provider.
func Shutdown(ctx context.Context) error {
	return gotrace.Shutdown(ctx)
}

// IsInitialized reports whether tracing is initialized.
func IsInitialized() bool {
	return gotrace.IsInitialized()
}
