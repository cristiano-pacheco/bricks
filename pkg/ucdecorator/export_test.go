package ucdecorator

import (
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
)

func NewTestFactory(cfg Config, m metrics.UseCaseMetrics, log logger.Logger, t ErrorTranslator) *Factory {
	return &Factory{cfg: cfg, metrics: m, logger: log, translator: t}
}

func (f *Factory) InferUseCaseName(handler any) string {
	return f.inferUseCaseName(handler)
}

func (f *Factory) InferMetricName(name string) string {
	return f.inferMetricName(name)
}

func (f *Factory) ToSnakeCase(value string) string {
	return f.toSnakeCase(value)
}

func WithLogging[T, R any](handler UseCase[T, R], log logger.Logger, name string) UseCase[T, R] {
	return withLogging(handler, log, name)
}

func WithMetrics[T, R any](handler UseCase[T, R], m metrics.UseCaseMetrics, name string) UseCase[T, R] {
	return withMetrics(handler, m, name)
}

func WithTracing[T, R any](handler UseCase[T, R], name string) UseCase[T, R] {
	return withTracing(handler, name)
}

func WithTranslation[T, R any](handler UseCase[T, R], t ErrorTranslator) UseCase[T, R] {
	return withTranslation(handler, t)
}

func WithDebug[T, R any](handler UseCase[T, R], log logger.Logger, useCaseName, decoratorName string) UseCase[T, R] {
	return withDebug(handler, log, useCaseName, decoratorName)
}
