package trace

import oteltrace "github.com/cristiano-pacheco/go-otel/trace"

const (
	ExporterTypeGRPC = oteltrace.ExporterTypeGRPC
	ExporterTypeHTTP = oteltrace.ExporterTypeHTTP
)

type ExporterType = oteltrace.ExporterType

func NewExporterType(value string) (ExporterType, error) {
	return oteltrace.NewExporterType(value)
}
