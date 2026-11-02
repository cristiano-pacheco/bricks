package trace

import oteltrace "github.com/cristiano-pacheco/go-otel/trace"

var (
	ErrAppNameRequired        = oteltrace.ErrAppNameRequired
	ErrTraceURLRequired       = oteltrace.ErrTraceURLRequired
	ErrInvalidLimits          = oteltrace.ErrInvalidLimits
	ErrInvalidSampleRate      = oteltrace.ErrInvalidSampleRate
	ErrInvalidExporterType    = oteltrace.ErrInvalidExporterType
	ErrAlreadyInitialized     = oteltrace.ErrAlreadyInitialized
	ErrNotInitialized         = oteltrace.ErrNotInitialized
	ErrCreateTracerProvider   = oteltrace.ErrCreateTracerProvider
	ErrCreateExporter         = oteltrace.ErrCreateExporter
	ErrCreateGRPCExporter     = oteltrace.ErrCreateGRPCExporter
	ErrCreateHTTPExporter     = oteltrace.ErrCreateHTTPExporter
	ErrTracerProviderShutdown = oteltrace.ErrTracerProviderShutdown
	ErrExporterShutdown       = oteltrace.ErrExporterShutdown
	ErrMultipleShutdown       = oteltrace.ErrMultipleShutdown
)
