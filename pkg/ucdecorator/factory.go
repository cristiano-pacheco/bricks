package ucdecorator

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
)

type Factory struct {
	metrics    metrics.UseCaseMetrics
	logger     logger.Logger
	translator ErrorTranslator
}

func NewFactory(
	useCaseMetrics metrics.UseCaseMetrics,
	log logger.Logger,
	translator ErrorTranslator,
) *Factory {
	return &Factory{
		metrics:    useCaseMetrics,
		logger:     log,
		translator: translator,
	}
}

func Wrap[T any, R any](
	factory *Factory,
	handler UseCase[T, R],
) UseCase[T, R] {
	useCaseName := factory.inferUseCaseName(handler)
	metricName := factory.inferMetricName(useCaseName)
	return Chain(handler, factory.logger, factory.metrics, factory.translator, metricName, useCaseName)
}

func (f *Factory) inferUseCaseName(handler any) string {
	t := reflect.TypeOf(handler)
	if t == nil {
		return "UseCase.Execute"
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeName := t.Name()
	if typeName == "" {
		return "UseCase.Execute"
	}

	return typeName + ".Execute"
}

func (f *Factory) inferMetricName(useCaseName string) string {
	name := strings.TrimSuffix(useCaseName, ".Execute")
	name = strings.TrimSuffix(name, "UseCase")
	if name == "" {
		return "use_case"
	}

	return f.toSnakeCase(name)
}

func (f *Factory) toSnakeCase(value string) string {
	const additionalCapacity = 4

	if value == "" {
		return ""
	}

	runes := []rune(value)
	out := make([]rune, 0, len(runes)+additionalCapacity)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				out = append(out, '_')
			}
		}
		out = append(out, unicode.ToLower(r))
	}

	return string(out)
}
