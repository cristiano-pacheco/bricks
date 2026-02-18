package ucdecorator

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
)

type Factory struct {
	cfg        Config
	metrics    metrics.UseCaseMetrics
	logger     logger.Logger
	translator ErrorTranslator
}

func NewFactory(
	cfg config.Config[Config],
	useCaseMetrics metrics.UseCaseMetrics,
	log logger.Logger,
	translator ErrorTranslator,
) *Factory {
	return &Factory{
		cfg:        cfg.Get(),
		metrics:    useCaseMetrics,
		logger:     log,
		translator: translator,
	}
}

func Wrap[T any, R any](
	factory *Factory,
	handler UseCase[T, R],
) UseCase[T, R] {
	cfg := factory.cfg
	if !cfg.Enabled {
		return handler
	}

	useCaseName := factory.inferUseCaseName(handler)
	metricName := factory.inferMetricName(useCaseName)

	result := handler

	if cfg.Translation {
		result = withTranslation(result, factory.translator)
		if cfg.DebugMode {
			factory.logger.Debug("applying translation decorator", logger.String("use_case", useCaseName))
		}
	}
	if cfg.Tracing {
		result = withTracing(result, useCaseName)
		if cfg.DebugMode {
			factory.logger.Debug("applying tracing decorator", logger.String("use_case", useCaseName))
		}
	}
	if cfg.Metrics {
		result = withMetrics(result, factory.metrics, metricName)
		if cfg.DebugMode {
			factory.logger.Debug("applying metrics decorator", logger.String("use_case", useCaseName))
		}
	}
	if cfg.Logging {
		result = withLogging(result, factory.logger, useCaseName)
		if cfg.DebugMode {
			factory.logger.Debug("applying logging decorator", logger.String("use_case", useCaseName))
		}
	}

	return result
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
